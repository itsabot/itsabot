# Abot
 
Abot is a digital assistant framework that enables anyone to easily build a digital assistant similar to Apple's Siri, Microsoft's Cortana, Google Now, or Amazon Echo. Further, Abot supports a human-aided training backend enabling anyone to build services like Facebook M or Magic.

Unlike those proprietary systems, Abot is open-sourced and extensible. By providing an extensible platform onto which anyone can easily add functionality, Abot is the first A.I. framework that aims to be available everywhere and—ultimately—to do everything.

*Note:* This is pre-alpha software and shouldn't yet be considered for production use.

## Architecture

Abot's core consists of three parts:

1. An API that accepts natural language inputs.
1. A state machine that tracks grammar and context across inputs, enabling the chaining of commands.
1. A router that selects the appropriate packages to send the input based on the current command and past context.

It combines those three parts with tools and libraries designed to make programming a digital assistant as quick and fun as possible.

Abot uses packages to easily extend functionality. If you want additional features, simply add the appropriate package to `packages.json` or build your own with the libraries and examples provided.

## License

Abot is MIT-licensed, a copy of which you can find in the repo.

The Abot logo is courtesy of Edward Boatman via TheNounProject and licensed via
Creative Commons Attribution v3.
