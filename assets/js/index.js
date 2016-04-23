(function(abot) {
abot.Index = {}
abot.Index.controller = function() {
	if (abot.isAdmin()) {
		return m.route("/admin")
	}
}
abot.Index.view = function() {
	return m(".container", [
		m.component(abot.Header),
		m(".main-no-sidebar", [
			m(".centered", [
				m("img[src=/public/images/logo.svg].big-icon"),
				m("h1", "Congratulations, you've set up Abot!"),
			]),
			m("p", "As next steps, try:"),
			m("ul", [
				m("li", [
					"Reading the ",
					m("a[href=https://github.com/itsabot/abot/wiki/Getting-Started]", "Getting Started guide.")
				]),
				m("li", [
					m("a[href=https://github.com/itsabot/abot/wiki/Getting-Started#communicating-with-abot]", "Communicating with Abot.")
				]),
				m("li", m("a[href=/signup]", {
					config: m.route,
				}, "Creating an Account.")),
				m("li", m("a[href=https://github.com/itsabot/abot/wiki/Adding-SMS-Messaging-to-Abot]", "Adding SMS Messaging to Abot.")),
				m("li", m("a[href=#/]", "Building a plugin.")),
				m("li", [
					"Learning the ",
					m("a[href=https://github.com/itsabot/abot/wiki/Learning-the-Human-Aided-Training-Interface]", "Human-Aided Training Interface.")
				]),
				m("li", [
					"Learning ",
					m("a[href=https://github.com/itsabot/abot/wiki/How-to-Contribute]", "How to Contribute.")
				]),
				m("li", [
					m("a[href=https://github.com/itsabot/abot/wiki/Getting-Started#deploying-your-abot]", "Deploying to Heroku."),
				])
			])
		])
	])
}
})(!window.abot ? window.abot={} : window.abot);
