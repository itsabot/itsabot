(function(abot) {
abot.PluginIcon = {}
abot.PluginIcon.controller = function(attrs) {
	var ctrl = this
	ctrl.viewPlugin = function() {
	}
}
abot.PluginIcon.view = function(ctrl, attrs) {
	attrs.Icon = attrs.Icon || "/public/images/icon_missing.svg"
	return m(".plugin-icon", [
		m("img", {
			alt: attrs.Name + " icon",
			src: "/public/images/" + attrs.AdminPanel.Icon,
		}),
		m(".name", attrs.Name),
	])
}
})(!window.abot ? window.abot={} : window.abot);
