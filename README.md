![Abot](http://i.imgur.com/WBACSyP.png)

[![Join the chat at https://gitter.im/itsabot/abot](https://badges.gitter.im/itsabot/abot.svg)](https://gitter.im/itsabot/abot?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

[Website](https://www.itsabot.org) |
[Getting Started](https://github.com/itsabot/abot/wiki/Getting-Started) |
[Contributing](https://github.com/itsabot/abot/wiki/How-to-Contribute) |
[Mailing List](https://groups.google.com/forum/#!forum/abot-discussion)

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/itsabot/abot) [![Travis CI](https://img.shields.io/travis/itsabot/abot.svg?style=flat-square)](https://travis-ci.org/itsabot/abot)
 
Abot (pronounced *Eh-Bot*, like the Canadians) is a digital assistant framework
that enables anyone to easily build a digital assistant similar to Apple's Siri,
Microsoft's Cortana, Google Now, or Amazon Alexa. Further, Abot supports a
human-aided training backend enabling anyone to build services like Facebook M.

Unlike those proprietary systems, Abot is open-sourced and extensible. By
providing an extensible platform onto which anyone can easily add functionality,
Abot is the first A.I. framework that aims to be available everywhere and—
ultimately—to do everything.

**Note: This is being developed heavily.** There may be breaking API changes
in each release until we hit v1.0. Follow our progress on the
[Roadmap](https://github.com/itsabot/abot/wiki/Roadmap).

## Installation

> **Dependencies**: Abot requires that the following programs are installed:
>
> * [Go](https://golang.org/dl/) >= 1.6
> * [PostgreSQL](http://www.postgresql.org/download/) >= 9.5

Fetch Abot via `go get`

```
$ go get github.com/itsabot/abot
```

Then create a new project anywhere in your `$GOPATH`, passing in your Postgres
credentials/host if needed. Projects should be named with camelCasing.

```
$ abot new yourproject [username[:password]@host[:port]]
Success! Created yourproject
```

If you don't pass anything to the command, the Postgres parameters will default
to `host = 127.0.0.1`, `port = 5432`, and `username = postgres`.  You may need
to edit your
[pg_hba.conf](http://www.postgresql.org/docs/9.5/static/auth-pg-hba-conf.html)
file if you want to use this password-less default.

During setup, if the `psql` binary is unavailable, the script will skip the
database setup. To setup the database on an different machine, you can run
`cmd/dbsetup.sh` on the host that has Postgres / `psql` available. This script
takes the same Postgres parameter as `abot new`.

Once the script completes, launch the server:

```
$ cd yourproject
$ abot server
```

Then visit Abot at `localhost:4200`.

## Usage

First configure the plugins you want to import, such as `weather`. Add them
to your plugins.json like so:

```json
{
	"Version": 0.2,
	"Dependencies": {
		"github.com/itsabot/plugin_weather": "*"
	}
}
```

Then run the following in your terminal to download the plugins:

```bash
$ abot install
Fetching 1 plugin...
Installing plugin...
Success!
```

That will download the plugins into your `$GOPATH` and install them into your
project.  Once you've installed the plugins, boot the server again: `abot
server`. You can then use the included Abot console to communicate with Abot
locally:

```bash
$ abot console
> Hi
Hello there!
```

You can learn more in our
[Getting Started](https://github.com/itsabot/abot/wiki/Getting-Started) guide.

## Goals

We believe that A.I. will impact every business worldwide and dramatically
change our lives. While Apple, Google and others rush to build proprietary
digital assistants, there's a great need for an open approach that can be made
to run anywhere and be customized to do anything you need.

Abot enables any person or business to build digital assistants like Siri using
plugins that are as easy to install and run as WordPress. Soon it'll be as easy
to leverage A.I. in your business as it is to start a blog or an online store.
Imagine setting up an AI assistant to answer your phones, schedule meetings,
and book travel for your company in 30 seconds or less. The future's almost
here, and Abot's going to lead the way.

We have a long road ahead of us, but "nothing ever comes to one that is worth
having except as a result of hard work." *-- Booker T. Washington*

Follow our progress on our
[Roadmap](https://github.com/itsabot/abot/wiki/Roadmap) or learn how you can
get involved with our
[Contributor's Guide](https://github.com/itsabot/abot/wiki#contributing).

## License

MIT, a copy of which you can find in the repo.

This project uses a Bayesian classifier library (github.com/jbrukh/bayesian),
whose BSD-style license you can find in `/core/training/LICENSE.md`.
