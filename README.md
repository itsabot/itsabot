# Abot [![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/itsabot/abot)

[Website](https://www.itsabot.org) | [Getting Started](https://github.com/itsabot/abot/wiki/Getting-Started) | [Contributing](https://github.com/itsabot/abot/wiki/How-to-Contribute) | [Mailing List](https://groups.google.com/forum/#!forum/abot-discussion)
 
Abot (pronounced *Eh-Bot*, like the Canadians) is a digital assistant framework
that enables anyone to easily build a digital assistant similar to Apple's Siri,
Microsoft's Cortana, Google Now, or Amazon Alexa. Further, Abot supports a
human-aided training backend enabling anyone to build services like Facebook M
or Magic.

Unlike those proprietary systems, Abot is open-sourced and extensible. By
providing an extensible platform onto which anyone can easily add functionality,
Abot is the first A.I. framework that aims to be available everywhere and—
ultimately—to do everything.

**Note: This is pre-alpha software and shouldn't yet be considered for
production use.**

## Installation

> **Dependencies**: Abot requires that the following programs are installed:
>
> * [Go](https://golang.org/dl/) >= 1.6
> * [PostgreSQL](http://www.postgresql.org/download/) >= 9.5

You can install Abot via git:

```
git clone git@github.com:itsabot/abot.git && cd abot
go install ./...
ABOT_PORT=4200 abot -s
```

Then visit Abot at `localhost:4200`.

## Usage

First configure the plugins you want to import, such as `restaurants` or
`mechanic`. Add them to your plugin.json like so:

```
{
	"Name": "abot",
	"Version": "0.0.1",
	"Dependencies": {
		"github.com/itsabot/plugin_onboard": "*",
		"github.com/itsabot/plugin_restaurants": "*",
		"github.com/itsabot/plugin_mechanic": "*"
	}
}
```

Then run the following in your terminal to download the plugins:

```
$ abotp
Fetching 3 plugins...
Success!
```

That will download the plugins to the `/plugins` directory. Be sure to follow
the integration instructions in the README of each plugin you add (found in
`/plugins/pluginname/README.md`), as adding a plugin may require you to make some
minor code changes in Abot. Once you've integrated the plugins, recompile and
run Abot again: `go install ./... && abot -s`. You can use the included Abot
console to communicate with Abot locally:

```
$ abotc
> Hi
Hello there!
```

You can learn more in our
[Getting Started](https://github.com/itsabot/abot/wiki/Getting-Started) guide.

## Goals

Abot aspires to:

1. Be accessible from every major communication method (SMS, email, web, as well
as native iOS and Android apps).
1. Open-source fully automated plugins covering common use-cases, like checking
the weather, stocks, restaurant recommendations, and more.
1. Distribute and share training efforts of the Abot community with opt-in
centralized training.
1. Offer simple deployment and hosting options for Abots with pre-configured
defaults.

## License

MIT, a copy of which you can find in the repo.

The Abot logo is courtesy of
[Edward Boatman](https://thenounproject.com/edward/) via TheNounProject and
licensed via Creative Commons Attribution v3.

The default plugin icon (puzzle piece) is courtesy of
[Arthur Shlain](https://thenounproject.com/ArtZ91/) via TheNounProject and
licensed via Creative Commons Attribution v3.
