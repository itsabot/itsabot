(function(abot) {
abot.PluginIcon = {}
abot.PluginIcon.controller = function(attrs) {
	var ctrl = this
	ctrl.loadDefault = function(ev) {
		ev.target.setAttribute("src", "/public/images/icon_missing.svg")
	}
	ctrl.iconName = function(name) {
		name = name.toLowerCase()
		return name.replace(" ", "_")
	}
}
abot.PluginIcon.view = function(ctrl, attrs) {
	attrs.Icon = attrs.Icon || "/public/images/icon_missing.svg"
	var img = m("div", [
		m("img", {
			alt: attrs.Name + " icon",
			src: "/public/images/icon_" + ctrl.iconName(attrs.Name) + ".svg",
			onerror: ctrl.loadDefault,
		}),
		m(".name", attrs.Name),
	])
	return m(".plugin-icon", [
		function() {
			if (!!attrs.HomeRoute) {
				return m("a", { href: attrs.HomeRoute, config: m.route }, img)
			}
			return img
		}(),
	])
}
})(!window.abot ? window.abot={} : window.abot);
