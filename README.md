# Ava

Ava is a general-purpose A.I. platform similar to Apple's Siri, Microsoft's Cortana, Google Now, or Amazon Echo.  Unlike those proprietary systems, Ava is open-sourced and extensible. By providing an extensible platform onto which anyone can easily add functionality, Ava is the first A.I. platform that aims to be available everywhere and—ultimately—to do everything.

## Architecture

Ava's core consists of three parts:

1. An API that accepts natural language inputs.
1. A state machine that tracks grammar and context across inputs, enabling the chaining of commands.
1. A router that selects the appropriate packages to send the input based on the current command and past context.

## About Packages

Packages provide all of Ava's functionality. Further, they may depend on one another.

## Setup

1. Install Postgres
1. Set up your environment vars (see below)
1. Get credentials for Yelp, Twilio, AWS, Stripe, Bonsai ElasticSearch, USPS, SendGrid, Rollbar, and Google API access
1. (Optional) Install npm and the npm package "rebuild". Use the following alias to automatically watch and cat your JS files

	alias watch="rebuild -w assets/js \"cat assets/js/*.js > public/js/main.js\""

	export BASE_URL="https://www.avabot.co/"
	export PORT="4200"
	export YELP_CONSUMER_KEY=""
	export YELP_CONSUMER_SECRET=""
	export YELP_TOKEN=""
	export YELP_TOKEN_SECRET=""
	export TWILIO_ACCOUNT_SID=""
	export TWILIO_AUTH_TOKEN=""
	export AWS_ACCESS_KEY_ID=""
	export AWS_SECRET_ACCESS_KEY=""
	export STRIPE_PUBLIC_KEY=""
	export STRIPE_ACCESS_TOKEN=""
	export ELASTICSEARCH_USERNAME=""
	export ELASTICSEARCH_PASSWORD=""
	export ELASTICSEARCH_DOMAIN=""
	export USPS_USER_ID=""
	export SENDGRID_KEY=""
	export ROLLBAR_ACCESS_TOKEN=""
	export ROLLBAR_ENDPOINT=""
	export GOOGLE_CLIENT_ID=""
	export GOOGLE_CLIENT_SECRET=""

## Installing and Running

`go install ./... && ava -s`
