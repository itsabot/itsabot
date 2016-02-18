(function(ava) {
ava.Index = {}
ava.Index.view = function() {
	return m(".main", [
		m.component(ava.Header),
		m(".centered", [
			m("img[src=/public/images/logo.svg].big-icon"),
			m("h1", "Congratulations, you've set up Abot!"),
		]),
		m("p", "As next steps, try:"),
		m("ul", [
			m("li", m("a[href=#/]", "Creating an Account")),
			m("li", m("a[href=#/]", "Speaking to Abot")),
			m("li", [
				"Reading the ",
				m("a[href=#/]", "Getting Started guide.")
			]),
			m("li", m("a[href=#/]", "Building a package")),
			m("li", [
				"Learning ",
				m("a[href=#/]", "How to Contribute.")
			]),
			m("li", "Deploying to Heroku (coming soon)")
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
