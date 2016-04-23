(function(abot) {
abot.Admin = {}
abot.Admin.controller = function() {
	var ctrl = this
	ctrl.props = {
		plugins: m.prop([])
	}
	abot.Plugins.fetch().then(function(resp) {
		ctrl.props.plugins(resp || [])
	}, function(err) {
		console.error(err)
	})
}
abot.Admin.view = function(ctrl) {
	return m(".container", [
		m.component(abot.Header),
		m.component(abot.Sidebar, { active: 0 }),
		m(".main", [
			m(".topbar", "Plugins"),
			m(".content", [
				m("h3.top-el", "Getting Started"),
				m("div", [
					"Looking for plugins? You can search and install plugins built by our community through ",
					m("a[href=https://www.itsabot.org/plugins]", "itsabot.org/plugins"),
					".",
				]),
				m("h3", "Installed Plugins"),
				function() {
					if (ctrl.props.plugins().length === 0) {
						return m("div", "No plugins installed.")
					}
				}(),
				ctrl.props.plugins().map(function(plugin) {
					return m.component(abot.PluginIcon, plugin)
				}),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
