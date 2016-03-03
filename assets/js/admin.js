(function(abot) {
abot.Admin = {}
abot.Admin.controller = function() {
	var ctrl = this
	abot.request({
		method: "GET",
		url: "/api/admin/plugins.json",
	}).then(function(resp) {
		console.log(resp)
		ctrl.props.installedPlugins(resp.Plugins)
	}, function(err) {
		console.error(err)
	})
	ctrl.props = {
		installedPlugins: m.prop([]),
	}
}
abot.Admin.view = function(ctrl) {
	return m(".main", [
		m.component(abot.Header),
		m("h1", "Admin Panel"),
		m("div", [
			m("h2", "Installed plugins"),
			function() {
				if (ctrl.props.installedPlugins().length === 0) {
					return m("div", "Plugins you've installed will be listed here.")
				}
			}(),
			ctrl.props.installedPlugins().map(function(plugin) {
				return m.component(abot.PluginIcon, plugin)
			}),
			m("h2", "Get plugins"),
			m("div", [
				"Looking for plugins? You can search and install plugins built by our community through ",
				m("a[href=https://www.itsabot.org/plugins]", "itsabot.org/plugins"),
				".",
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
