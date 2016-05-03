(function(abot) {
abot.Sidebar = {}
abot.Sidebar.view = function(_, args) {
	return m(".sidebar", [
		m("ul", [
			m.component(abot.SidebarItem, {
				href: "/admin",
				text: "Plugins",
				active: args.active === 0,
			}),
			m.component(abot.SidebarItem, {
				href: "/training",
				text: "Training",
				active: args.active === 1,
			}),
			/*
			m.component(abot.SidebarItem, {
				href: "#/",
				text: "Response Panel",
				active: args.active === 2,
			}),
			*/
			m.component(abot.SidebarItem, {
				href: "/manage_team",
				text: "Manage Team",
				active: args.active === 3,
			}),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
