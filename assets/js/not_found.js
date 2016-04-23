(function(abot) {
abot.NotFound = {}
abot.NotFound.view = function() {
	return m(".container", [
		m.component(abot.Header),
		m(".main-no-sidebar", [
			m(".centered.content", [
				m("h1", "404 - Not Found"),
				m("p", "That page doesn't seem to exist."),
				m("div", m("a[href=/]", { config: m.route }, "Go home")),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
