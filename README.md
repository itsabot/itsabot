# Ava

Ava is a general-purpose A.I. platform similar to Apple's Siri, Microsoft's Cortana, Google Now, or Amazon Echo.  Unlike those proprietary systems, Ava is open-sourced and extensible. By providing an extensible platform onto which anyone can easily add functionality, Ava is the first A.I. platform that aims to be available everywhere and—ultimately—to do everything.

## Architecture

Ava's core consists of three parts:

1. An API that accepts natural language inputs.
1. A state machine that tracks grammar and context across inputs, enabling the chaining of commands.
1. A router that selects the appropriate packages to send the input based on the current command and past context.

## About Packages

Packages provide all of Ava's functionality. Further, they may depend on one another.
