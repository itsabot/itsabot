(function(abot) {
abot.Dashboard = {}
abot.Dashboard.controller = function() {
	var ctrl = this
	ctrl.refresh = function() {
		abot.request({
			url: "/api/admin/dashboard.json",
			method: "GET",
		}).then(function(resp) {
			ctrl.props.checklist(resp.Checklist)
			ctrl.props.users(resp.Users.toLocaleString())
			ctrl.props.messages(resp.Messages.toLocaleString())
			ctrl.props.automationRate(resp.AutomationRate)
			ctrl.props.needUpdate(resp.NeedUpdate)
		})
	}
	ctrl.props = {
		plugins: m.prop([]),
		checklist: m.prop([]),
		users: m.prop(""),
		messages: m.prop(""),
		automationRate: m.prop(0.0),
		needUpdate: m.prop(false),

	}
	abot.Plugins.fetch().then(function(resp) {
		ctrl.props.plugins(resp || [])
	}, function(err) {
		console.error(err)
	})
	ctrl.refresh()
}
abot.Dashboard.view = function(ctrl) {
	return m(".body", [
		m.component(abot.Header),
		m(".container", [
			m.component(abot.Sidebar, { active: 0 }),
			m(".main", [
				m(".topbar", "Dashboard"),
				m(".content", [
					function() {
						if (!ctrl.props.needUpdate()) {
							return
						}
						return m(".well.well-full.alert.alert-warn", [
							"Abot update available. ",
							m("a[href=https://github.com/itsabot/abot/commit/master]", "See what's new."),
						])
					}(),
					m(".well.well-full", [
						m(".well-padding", [
							m(".well-header", "Setup Checklist"),
							m(".well-content", [
								function() {
									var els = []
									var cs = ctrl.props.checklist();
									for (var i = 0; i < cs.length; ++i) {
										var title = ""
										switch (i) {
										case 0:
											title = "Create admin"
											break
										case 1:
											title = "Install a plugin"
											break
										case 2:
											title = "Configure email or SMS"
											break
										case 3:
											title = "Connect your account"
											break
										case 4:
											title = "Invite another admin"
											break
										}
										els.push(m("div", [
											m("input[type=checkbox]", {
												checked: cs[i],
												disabled: true,
											}),
											title
										]))
									}
									return els
								}(),
							]),
						]),
					]),
					m(".well.well-sm", [
						m(".well-padding", [
							m(".well-header", "Users"),
							m(".well-content-lg", ctrl.props.users()),
						]),
					]),
					m(".well.well-sm", [
						m(".well-padding", [
							m(".well-header", "Messages"),
							m(".well-content-lg", ctrl.props.messages()),
						]),
					]),
					m(".well.well-sm", [
						m(".well-padding", [
							m(".well-header", "Automation Rate"),
							m(".well-content-lg", ctrl.props.automationRate().toPrecision(2) * 100 + "%"),
						]),
					]),
					m(".well.well-sm", [
						m(".well-padding", [
							m(".well-header", "Installed Plugins"),
							m(".well-content.centered", [
								ctrl.props.plugins().map(function(plugin) {
									return m.component(abot.PluginIcon, plugin)
								}),
							]),
						]),
						m(".well-footer", [
							m("a[href=https://www.itsabot.org/plugins]", "Get Plugins"),
						]),
					]),
				]),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
