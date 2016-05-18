(function(abot) {
abot.Sidebar = {}
abot.Sidebar.view = function(_, args) {
	return m(".sidebar", [
		m("ul", [
			m.component(abot.SidebarItem, {
				href: "/admin",
				text: "Plugins",
				active: args.active === 0,
				icon: "grid.svg",
			}),
			m.component(abot.SidebarItem, {
				href: "/training",
				text: "Training",
				active: args.active === 1,
				icon: "flash.svg",
			}),
			m.component(abot.SidebarItem, {
				href: "/response_panel",
				text: "Response Panel",
				active: args.active === 2,
				icon: "message smile.svg",
			}),
			m.component(abot.SidebarItem, {
				href: "/manage_team",
				text: "Manage Team",
				active: args.active === 3,
				icon: "users.svg",
			}),
			m.component(abot.SidebarItem, {
				href: "/account_connect",
				text: "Account Connect",
				active: args.active === 4,
				icon: "link.svg",
			}),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
