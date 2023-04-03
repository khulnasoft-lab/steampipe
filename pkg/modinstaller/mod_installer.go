package modinstaller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver"
	git "github.com/go-git/go-git/v5"
	"github.com/otiai10/copy"
	"github.com/spf13/viper"
	"github.com/turbot/steampipe/pkg/constants"
	"github.com/turbot/steampipe/pkg/error_helpers"
	"github.com/turbot/steampipe/pkg/filepaths"
	"github.com/turbot/steampipe/pkg/steampipeconfig/modconfig"
	"github.com/turbot/steampipe/pkg/steampipeconfig/parse"
	"github.com/turbot/steampipe/pkg/steampipeconfig/versionmap"
	"github.com/turbot/steampipe/pkg/utils"
)

type ModInstaller struct {
	installData  *InstallData
	workspaceMod *modconfig.Mod
	mods         versionmap.VersionConstraintMap

	// the final resting place of all dependency mods
	modsPath string
	// temp location used to install dependencies
	tmpPath string
	// a shadow directory for installing mods
	// this is necessary to make mod installation transactional
	shadowDirPath string

	workspacePath string

	// what command is being run
	command string
	// are dependencies being added to the workspace
	dryRun bool
}

func NewModInstaller(opts *InstallOpts) (*ModInstaller, error) {
	i := &ModInstaller{
		workspacePath: opts.WorkspacePath,
		command:       opts.Command,
		dryRun:        opts.DryRun,
	}
	if err := i.setModsPath(); err != nil {
		return nil, err
	}

	// load workspace mod, creating a default if needed
	workspaceMod, err := i.loadModfile(i.workspacePath, true)
	if err != nil {
		return nil, err
	}
	i.workspaceMod = workspaceMod

	// load lock file
	workspaceLock, err := versionmap.LoadWorkspaceLock(i.workspacePath)
	if err != nil {
		return nil, err
	}

	// create install data
	i.installData = NewInstallData(workspaceLock, workspaceMod)

	// parse args to get the required mod versions
	requiredMods, err := i.GetRequiredModVersionsFromArgs(opts.ModArgs)
	if err != nil {
		return nil, err
	}
	i.mods = requiredMods

	return i, nil
}

func (i *ModInstaller) setModsPath() error {
	dir, err := os.MkdirTemp(os.TempDir(), "sp_dr_*")
	if err != nil {
		return err
	}
	i.tmpPath = dir
	i.modsPath = filepaths.WorkspaceModPath(i.workspacePath)
	i.shadowDirPath = filepaths.WorkspaceModShadowPath(i.workspacePath)
	return nil
}

func (i *ModInstaller) UninstallWorkspaceDependencies() error {
	workspaceMod := i.workspaceMod

	// remove required dependencies from the mod file
	if len(i.mods) == 0 {
		workspaceMod.RemoveAllModDependencies()

	} else {
		// verify all the mods specifed in the args exist in the modfile
		workspaceMod.RemoveModDependencies(i.mods)
	}

	// uninstall by calling Install
	if err := i.installMods(workspaceMod.Require.Mods, workspaceMod); err != nil {
		return err
	}

	if workspaceMod.Require.Empty() {
		workspaceMod.Require = nil
	}

	// if this is a dry run, return now
	if i.dryRun {

		log.Printf("[TRACE] UninstallWorkspaceDependencies - dry-run=true, returning before saving mod file and cache\n")
		return nil
	}

	// write the lock file
	if err := i.installData.Lock.Save(); err != nil {
		return err
	}

	//  now safe to save the mod file
	if err := i.workspaceMod.Save(); err != nil {
		return err
	}

	// tidy unused mods
	if viper.GetBool(constants.ArgPrune) {
		if _, err := i.Prune(); err != nil {
			return err
		}
	}
	return nil
}

// InstallWorkspaceDependencies installs all dependencies of the workspace mod
func (i *ModInstaller) InstallWorkspaceDependencies() (err error) {
	workspaceMod := i.workspaceMod
	defer func() {
		// tidy unused mods
		// (put in defer so it still gets called in case of errors)
		if viper.GetBool(constants.ArgPrune) {
			// be sure not to overwrite an existing return error
			_, pruneErr := i.Prune()
			if pruneErr != nil && err == nil {
				err = pruneErr
			}
		}
	}()

	// first check our Steampipe version is sufficient
	if err := workspaceMod.Require.ValidateSteampipeVersion(workspaceMod.Name()); err != nil {
		return err
	}

	// if mod args have been provided, add them to the the workspace mod requires
	// (this will replace any existing dependencies of same name)
	if len(i.mods) > 0 {
		workspaceMod.AddModDependencies(i.mods)
	}

	if err := i.installMods(workspaceMod.Require.Mods, workspaceMod); err != nil {
		return err
	}

	// if this is a dry run, return now
	if i.dryRun {
		log.Printf("[TRACE] InstallWorkspaceDependencies - dry-run=true, returning before saving mod file and cache\n")
		return nil
	}

	// write the lock file
	if err := i.installData.Lock.Save(); err != nil {
		return err
	}

	//  now safe to save the mod file
	if len(i.mods) > 0 {
		if err := i.workspaceMod.Save(); err != nil {
			return err
		}
	}

	if !workspaceMod.HasDependentMods() {
		// there are no dependencies - delete the cache
		i.installData.Lock.Delete()
	}
	return nil
}

func (i *ModInstaller) GetModList() string {
	return i.installData.Lock.GetModList(i.workspaceMod.GetInstallCacheKey())
}

func (i *ModInstaller) installMods(mods []*modconfig.ModVersionConstraint, parent *modconfig.Mod) (err error) {
	defer func() {
		if err == nil {
			// everything went well
			// replace the mods directory with the mods directory
			os.RemoveAll(i.modsPath)
			os.Rename(i.shadowDirPath, i.modsPath)
		}
		// remove any temporary directory
		// TODO BINAEK :: now that we have a shadow directory,
		// we shouldn't need this. try to remove this
		// clean up the temp location
		os.RemoveAll(i.tmpPath)

		// force remove the shadow directory
		os.RemoveAll(i.shadowDirPath)
	}()

	var errors []error
	for _, requiredModVersion := range mods {
		// modToUse, err := i.getCurrentlyInstalledVersionToUse(requiredModVersion, parent, i.updating())
		// if err != nil {
		// 	errors = append(errors, err)
		// 	continue
		// }

		// // if the mod is not installed or needs updating, pass shouldUpdate=true into installModDependencesRecursively
		// // this ensures that we update any dependencies which have updates available
		// shouldUpdate := modToUse == nil
		if err := i.installModDependencesRecursively(requiredModVersion, parent); err != nil {
			errors = append(errors, err)
		}
	}

	// update the lock to be the new lock, and record any uninstalled mods
	i.installData.onInstallComplete()

	return i.buildInstallError(errors)
}

func (i *ModInstaller) buildInstallError(errors []error) error {
	if len(errors) == 0 {
		return nil
	}
	verb := "install"
	if i.updating() {
		verb = "update"
	}
	prefix := fmt.Sprintf("%d %s failed to %s", len(errors), utils.Pluralize("dependency", len(errors)), verb)
	err := error_helpers.CombineErrorsWithPrefix(prefix, errors...)
	return err
}

func (i *ModInstaller) installModDependencesRecursively(requiredModVersion *modconfig.ModVersionConstraint, parent *modconfig.Mod) error {
	// get available versions for this mod
	includePrerelease := requiredModVersion.Constraint.IsPrerelease()
	availableVersions, err := i.installData.getAvailableModVersions(requiredModVersion.Name, includePrerelease)

	if err != nil {
		return err
	}

	// get a resolved mod ref that satisfies the version constraints
	resolvedRef, err := i.getModRefSatisfyingConstraints(requiredModVersion, availableVersions)
	if err != nil {
		return err
	}

	// install the mod
	dependencyMod, err := i.install(resolvedRef, parent)
	if err != nil {
		return err
	}
	if err = dependencyMod.ValidateSteampipeVersion(); err != nil {
		return err
	}

	// to get here we have the dependency mod - either we installed it or it was already installed
	// recursively install its dependencies
	var errors []error
	// now update the parent to dependency mod and install its child dependencies
	parent = dependencyMod
	for _, dep := range dependencyMod.Require.Mods {
		// childDependencyMod, err := i.getCurrentlyInstalledVersionToUse(dep, parent, shouldUpdate)
		// if err != nil {
		// 	errors = append(errors, err)
		// 	continue
		// }
		if err := i.installModDependencesRecursively(dep, parent); err != nil {
			errors = append(errors, err)
			continue
		}
	}

	return error_helpers.CombineErrorsWithPrefix(fmt.Sprintf("%d child %s failed to install", len(errors), utils.Pluralize("dependency", len(errors))), errors...)
}

func (i *ModInstaller) getCurrentlyInstalledVersionToUse(requiredModVersion *modconfig.ModVersionConstraint, parent *modconfig.Mod, forceUpdate bool) (*modconfig.Mod, error) {
	// do we have an installed version of this mod matching the required mod constraint
	installedVersion, err := i.installData.Lock.GetLockedModVersion(requiredModVersion, parent)
	if err != nil {
		return nil, err
	}
	if installedVersion == nil {
		return nil, nil
	}

	// can we update this
	canUpdate, err := i.canUpdateMod(installedVersion, requiredModVersion, forceUpdate)
	if err != nil {
		return nil, err

	}
	if canUpdate {
		// return nil mod to indicate we should update
		return nil, nil
	}

	// load the existing mod and return
	return i.loadDependencyMod(installedVersion)
}

// determine if we should update this mod, and if so whether there is an update available
func (i *ModInstaller) canUpdateMod(installedVersion *versionmap.ResolvedVersionConstraint, requiredModVersion *modconfig.ModVersionConstraint, forceUpdate bool) (bool, error) {
	// so should we update?
	// if forceUpdate is set or if the required version constraint is different to the locked version constraint, update
	// TODO check * vs latest - maybe need a custom equals?
	if forceUpdate || installedVersion.Constraint != requiredModVersion.Constraint.Original {
		// get available versions for this mod
		includePrerelease := requiredModVersion.Constraint.IsPrerelease()
		availableVersions, err := i.installData.getAvailableModVersions(requiredModVersion.Name, includePrerelease)
		if err != nil {
			return false, err
		}

		return i.updateAvailable(requiredModVersion, installedVersion.Version, availableVersions)
	}
	return false, nil

}

// determine whether there is a newer mod version avoilable which satisfies the dependency version constraint
func (i *ModInstaller) updateAvailable(requiredVersion *modconfig.ModVersionConstraint, currentVersion *semver.Version, availableVersions []*semver.Version) (bool, error) {
	latestVersion, err := i.getModRefSatisfyingConstraints(requiredVersion, availableVersions)
	if err != nil {
		return false, err
	}
	if latestVersion.Version.GreaterThan(currentVersion) {
		return true, nil
	}
	return false, nil
}

// get the most recent available mod version which satisfies the version constraint
func (i *ModInstaller) getModRefSatisfyingConstraints(modVersion *modconfig.ModVersionConstraint, availableVersions []*semver.Version) (*ResolvedModRef, error) {
	// find a version which satisfies the version constraint
	var version = getVersionSatisfyingConstraint(modVersion.Constraint, availableVersions)
	if version == nil {
		return nil, fmt.Errorf("no version of %s found satisfying version constraint: %s", modVersion.Name, modVersion.Constraint.Original)
	}

	return NewResolvedModRef(modVersion, version)
}

// install a mod
func (i *ModInstaller) install(dependency *ResolvedModRef, parent *modconfig.Mod) (_ *modconfig.Mod, err error) {
	var modDef *modconfig.Mod
	// get the temp location to install the mod to
	fullName := dependency.FullName()
	tempDestPath := i.getDependencyTmpPath(fullName)

	defer func() {
		if err == nil {
			i.installData.onModInstalled(dependency, modDef, parent)
		}
	}()
	// if the target path exists, use the exiting file
	// if it does not exist (the usual case), install it
	if _, err := os.Stat(tempDestPath); os.IsNotExist(err) {
		if err := i.installFromGit(dependency, tempDestPath); err != nil {
			return nil, err
		}
	}

	// now load the installed mod and return it
	modDef, err = i.loadModfile(tempDestPath, false)
	if err != nil {
		return nil, err
	}
	if modDef == nil {
		return nil, fmt.Errorf("'%s' has no mod definition file", dependency.FullName())
	}

	// so we have successfully installed this dependency to the temp location, now copy to the mod location
	if !i.dryRun {
		destPath := i.getDependencyShadowPath(fullName)
		if err := i.copyModFromTempToModsFolder(tempDestPath, destPath); err != nil {
			return nil, err
		}
		// now the mod is installed in it's final location, set mod dependency path
		if err := i.setModDependencyConfig(modDef, destPath); err != nil {
			return nil, err
		}
	}

	return modDef, nil
}

func (i *ModInstaller) copyModFromTempToModsFolder(tmpPath string, destPath string) error {
	if err := os.RemoveAll(destPath); err != nil {
		return err
	}

	if err := copy.Copy(tmpPath, destPath); err != nil {
		return err
	}
	return nil
}

func (i *ModInstaller) installFromGit(dependency *ResolvedModRef, installPath string) error {
	// get the mod from git
	gitUrl := getGitUrl(dependency.Name)
	_, err := git.PlainClone(installPath,
		false,
		&git.CloneOptions{
			URL:           gitUrl,
			ReferenceName: dependency.GitReference,
			Depth:         1,
			SingleBranch:  true,
		})

	return err
}

// build the path of the temp location to copy this depednency to
func (i *ModInstaller) getDependencyTmpPath(dependencyFullName string) string {
	return filepath.Join(i.tmpPath, dependencyFullName)
}

// build the path of the temp location to copy this depednency to
func (i *ModInstaller) getDependencyDestPath(dependencyFullName string) string {
	return filepath.Join(i.modsPath, dependencyFullName)
}

// build the path of the temp location to copy this depednency to
func (i *ModInstaller) getDependencyShadowPath(dependencyFullName string) string {
	return filepath.Join(i.shadowDirPath, dependencyFullName)
}

func (i *ModInstaller) loadDependencyMod(modVersion *versionmap.ResolvedVersionConstraint) (*modconfig.Mod, error) {
	modPath := i.getDependencyDestPath(modconfig.BuildModDependencyPath(modVersion.Name, modVersion.Version))
	modDef, err := i.loadModfile(modPath, false)
	if err != nil {
		return nil, err
	}
	if modDef == nil {
		return nil, fmt.Errorf("failed to load mod from %s", modPath)
	}
	if err := i.setModDependencyConfig(modDef, modPath); err != nil {
		return nil, err
	}
	return modDef, nil

}

// set the mod dependency path
func (i *ModInstaller) setModDependencyConfig(mod *modconfig.Mod, modPath string) error {
	dependencyPath, err := filepath.Rel(i.modsPath, modPath)
	if err != nil {
		return err
	}
	return mod.SetDependencyConfig(dependencyPath)
}

func (i *ModInstaller) loadModfile(modPath string, createDefault bool) (*modconfig.Mod, error) {
	if !parse.ModfileExists(modPath) {
		if createDefault {
			mod := modconfig.CreateDefaultMod(i.workspacePath)
			return mod, nil
		}
		return nil, nil
	}

	mod, err := parse.ParseModDefinition(modPath)
	if err != nil {
		return nil, err
	}

	return mod, nil
}

func (i *ModInstaller) updating() bool {
	return i.command == "update"
}

func (i *ModInstaller) uninstalling() bool {
	return i.command == "uninstall"
}
