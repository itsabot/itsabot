# Abot [![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/itsabot/abot) [![Travis CI](https://img.shields.io/travis/itsabot/abot.svg?style=flat-square)](https://travis-ci.org/itsabot/abot)

[Website](https://www.itsabot.org) |
[Getting Started](https://github.com/itsabot/abot/wiki/Getting-Started) |
[Contributing](https://github.com/itsabot/abot/wiki/How-to-Contribute) |
[Mailing List](https://groups.google.com/forum/#!forum/abot-discussion)
 
Abot (pronounced *Eh-Bot*, like the Canadians) is a digital assistant framework
that enables anyone to easily build a digital assistant similar to Apple's Siri,
Microsoft's Cortana, Google Now, or Amazon Alexa. Further, Abot supports a
human-aided training backend enabling anyone to build services like Facebook M.

Unlike those proprietary systems, Abot is open-sourced and extensible. By
providing an extensible platform onto which anyone can easily add functionality,
Abot is the first A.I. framework that aims to be available everywhere and—
ultimately—to do everything.

**Note: This is pre-alpha software and shouldn't yet be considered for
production use.** Follow our progress on the
[Roadmap](https://github.com/itsabot/abot/wiki/Roadmap).

## Installation

> **Dependencies**: Abot requires that the following programs are installed:
>
> * [Go](https://golang.org/dl/) >= 1.6
> * [PostgreSQL](http://www.postgresql.org/download/) >= 9.5

You can install Abot via git:

```bash
$ git clone git@github.com:itsabot/abot.git && cd abot && chmod +x cmd/*.sh
$ cmd/setup.sh
$ abot server
```

Then visit Abot at `localhost:4200`.

## Usage

First configure the plugins you want to import, such as `restaurants` or
`mechanic`. Add them to your plugin.json like so:

```json
{
	"Name": "abot",
	"Version": "0.0.1",
	"Dependencies": {
		"github.com/itsabot/plugin_onboard": "*",
		"github.com/itsabot/plugin_restaurants": "*",
		"github.com/itsabot/plugin_weather": "*"
	}
}
```

Then run the following in your terminal to download the plugins:

```bash
$ abot plugin install
Fetching 3 plugins...
Success!
```

That will download the plugins to the `/plugins` directory. Once you've
integrated the plugins, recompile and run Abot again: `abot server`. You can
use the included Abot console to communicate with Abot locally:

```bash
$ abot console +13105555555
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

The Abot logo is courtesy of
[Edward Boatman](https://thenounproject.com/edward/) via TheNounProject and
licensed via Creative Commons Attribution v3.

The default plugin icon (puzzle piece) is courtesy of
[Arthur Shlain](https://thenounproject.com/ArtZ91/) via TheNounProject and
licensed via Creative Commons Attribution v3.
