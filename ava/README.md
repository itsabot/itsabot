# Ava

Ava is a general-purpose A.I. platform similar to Apple's Siri, Microsoft's Cortana, Google Now, or Amazon Echo.  Unlike those proprietary systems, Ava is open-sourced and extensible. By providing an extensible platform onto which anyone can easily add functionality, Ava is the first A.I. platform that aims to be available everywhere and—ultimately—to do everything.

This is a lifelong project. A fifty year endeavor. My personal Sagrada Familia or Linux kernel programmed in my spare time.

## Architecture

Ava's core consists of three parts:

1. An API that accepts natural language inputs.
1. A state machine that tracks grammar and context across inputs, enabling the chaining of commands.
1. A router that selects the appropriate packages to send the input based on the current command and past context.

Supplemental to those parts are a number of tools, yet to be built.

1. A high-level machine learning library which abstracts away the complexities of NER, classification, etc. providing pragmatic solutions and trained models to common machine learning tasks. A standardized API provides a stable interface, abstracting away the underlying tools (e.g. MITIE vs Stanford's Open IE), and enabling a package developer to build Ava packages using state-of-the-art machine learning from academia with a simple and unchanging interface.
1. A package manager and central hosting service that installs and configures packages and dependencies, similar to `go install`. Packages can be built using any language that compiles down to an executable binary, but for compatibility with the suite of tools Ava will provide as well as eligibility to be hosted on the central hosting service, it's recommended that packages are built with Go. Packages may be closed source because packages are binary executables and *not* source-readable files.

## About Packages

Packages provide all of Ava's functionality. Further, they may depend on one another.

For instance, the meeting package (github.com/avabot/meeting) schedules meetings based on inputs. It requires the Ava gcal package to schedule meetings around your existing calendar. For email integration, additional Ava packages are also needed.

## TODO

1. HTTP endpoint for creating structured input, routing to packages
2. Example package
3. Package manager and storage server
