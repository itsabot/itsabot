(function(abot) {
abot.SidebarItem = {}
abot.SidebarItem.view = function(_, args) {
	var klass = "";
	if (args.active) {
		klass = "active"
	}
	return m("li", [
		m("a", {
			href: args.href,
			config: m.route,
			"class": klass,
		}, [
			m("img", { src: "/public/images/"+args.icon }),
			args.text,
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
