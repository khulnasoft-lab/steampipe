[<picture><source media="(prefers-color-scheme: dark)" srcset="https://steampipe.io/images/steampipe-color-logo-and-wordmark-with-white-bubble.svg"><source media="(prefers-color-scheme: light)" srcset="https://steampipe.io/images/steampipe-color-logo-and-wordmark-with-white-bubble.svg"><img width="67%" alt="Steampipe Logo" src="https://steampipe.io/images/steampipe-color-logo-and-wordmark-with-white-bubble.svg"></picture>](https://steampipe.io?utm_id=gspreadme&utm_source=github&utm_medium=repo&utm_campaign=github&utm_content=readme)

[![plugins](https://img.shields.io/badge/apis_supported-140-blue)](https://hub.powerpipe.io/plugins?utm_id=gspreadme&utm_source=github&utm_medium=repo&utm_campaign=github&utm_content=readme) &nbsp; 
[![slack](https://img.shields.io/badge/slack-2297-blue)](https://turbot.com/community/join?utm_id=gspreadme&utm_source=github&utm_medium=repo&utm_campaign=github&utm_content=readme) &nbsp;
[![maintained by](https://img.shields.io/badge/maintained%20by-Turbot-blue)](https://turbot.com?utm_id=gspreadme&utm_source=github&utm_medium=repo&utm_campaign=github&utm_content=readme)


[Steampipe](https://steampipe.io) is **the zero-ETL way** to query APIs and services. Use it to expose data sources to SQL.

We offer these Steampipe distributions:

**Steampipe CLI**. Run [queries](https://steampipe.io/docs/query/overview) that translate APIs to tables in the Postgres instance that's bundled with Steampipe.

**Steampipe Postgres FDWs**. Use [native Postgres Foreign Data Wrappers](https://steampipe.io/docs/steampipe_postgres/overview) to translate APIs to foreign tables.

**Steampipe SQLite extensions**. Use [SQLite extensions](https://steampipe.io/docs/steampipe_sqlite/overview) to translate APIS to SQLite virtual tables.

**Steampipe export tools**. Use [standalone binaries](https://steampipe.io/docs/steampipe_export/overview) that export data from APIs, no database required.

**Turbot Pipes**. Use [Turbot Pipes](https://turbot.com/pipes) to run Steampipe in the cloud.

## Demo time!

![](https://steampipe.io/images/steampipe-sql-demo.gif)

## Install Steampipe

 The <a href="https://steampipe.io/downloads?utm_id=gspreadme&utm_source=github&utm_medium=repo&utm_campaign=github&utm_content=readme">downloads</a> page shows you how but tl;dr:
 
Linux or WSL

```sh
sudo /bin/sh -c "$(curl -fsSL https://steampipe.io/install/steampipe.sh)"
```

MacOS

```sh
brew tap turbot/tap
brew install steampipe
```

Now, [install a plugin and run your first query →](https://steampipe.io/docs)

## Steampipe plugins

The Steampipe community has grown a suite of [plugins](https://hub.powerpipe.io/plugins?utm_id=gspreadme&utm_source=github&utm_medium=repo&utm_campaign=github&utm_content=readme) that map APIs to database tables. There are plugins for [AWS](https://hub.steampipe.io/plugins/turbot/aws), [Azure](https://hub.steampipe.io/plugins/turbot/azure), [GCP](https://hub.steampipe.io/plugins/turbot/gcp), [Kubernetes](https://hub.steampipe.io/plugins/turbot/kubernetes), [GitHub](https://hub.steampipe.io/plugins/turbot/github), [Microsoft 365](https://hub.steampipe.io/plugins/turbot/microsoft365), [Salesforce](https://hub.steampipe.io/plugins/turbot/salesforce) and many more.

Some plugins represent APIs as a handful of tables, others as hundreds of tables; the AWS leads the pack with over 450. There are more than 2000 tables in all, each clearly documented with copy/paste/run examples.

## Developing

<details>
 <summary>Prerequisites.</summary>
 - [Golang](https://golang.org/doc/install) Version 1.19 or higher.
</details>


<details>
 <summary>Clone the repo.</summary>
 
```
git clone https://github.com/turbot/steampipe
```
</details>

<details>
 <summary>Build.</summary>
 
 ```
 cd steampipe
 make
 ```

The build installs the new version to your `/usr/local/bin/` directory:
</details>

<details>
 <summary>
  Check the version.
 </summary>
 
 ```
$ steampipe -v
steampipe version 0.21.1
```
</details>

<details>
 <summary>Run a query.</summary>
 
  Now [install a plugin and run your first query →](https://steampipe.io/docs).
</details>

## Open source and contributing

This repository is published under the [AGPL 3.0](https://www.gnu.org/licenses/agpl-3.0.html) license. Please see our [code of conduct](https://github.com/turbot/.github/blob/main/CODE_OF_CONDUCT.md). Contributors must sign our [Contributor License Agreement](https://turbot.com/open-source#cla) as part of their first pull request. We look forward to collaborating with you!

[Steampipe](https://steampipe.io) is a product produced from this open source software, exclusively by [Turbot HQ, Inc](https://turbot.com). It is distributed under our commercial terms. Others are allowed to make their own distribution of the software, but cannot use any of the Turbot trademarks, cloud services, etc. You can learn more in our [Open Source FAQ](https://turbot.com/open-source).

## Turbot Pipes

 Bring your team to [Turbot Pipes](https://turbot.com/pipes?utm_id=gspreadme&utm_source=github&utm_medium=repo&utm_campaign=github&utm_content=readme) to use Steampipe together in the cloud.

## Get involved

**[Join #steampipe on Slack →](https://turbot.com/community/join)**

Want to help but don't know where to start? Pick up one of the `help wanted` issues:
* [Steampipe](https://github.com/turbot/steampipe/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22)
