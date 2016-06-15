(function(abot) {
abot.Settings = {}
abot.Settings.controller = function() {
	var ctrl = this
	ctrl.save = function() {
		var changed = document.querySelectorAll(".input-changed")
		if (changed.length === 0) {
			return
		}
		var data = {}
		var ps = ctrl.props.plugins()
		for (var i = 0; i < ps.length; ++i) {
			var p = ps[i]
			if (p.Settings == null) {
				continue
			}
			data[p.Name] = {}
			Object.keys(p.Settings).map(function(settingName) {
				data[p.Name][settingName] = p.Settings[settingName] || ""
			})
		}
		abot.request({
			url: "/api/admin/settings.json",
			method: "PUT",
			data: data,
		}).then(function(resp) {
			ctrl.props.error("")
			ctrl.props.success("Success! Saved changes.")
			for (var i = 0; i < changed.length; ++i) {
				changed[i].classList.remove("input-changed")
			}
		}, function(err) {
			ctrl.props.success("")
			ctrl.props.error(err.Msg)
		})
	}
	ctrl.markChanged = function(val) {
		this.classList.add("input-changed")
		var pluginIdx = this.getAttribute("data-plugin-idx")
		var setting = this.getAttribute("data-setting")
		ctrl.props.plugins()[pluginIdx].Settings[setting] = val
	}
	ctrl.discard = function() {
		if (document.querySelector(".input-changed") == null) {
			return
		}
		if (confirm("Are you sure you want to discard your changes?")) {
			m.route(window.location.pathname, null, true)
		}
	}
	ctrl.props = {
		plugins: m.prop([]),
		error: m.prop(""),
		success: m.prop(""),
	}
	abot.Plugins.fetch().then(function(resp) {
		ctrl.props.plugins(resp || [])
	}, function(err) {
		ctrl.props.error(err.Msg)
	})
}
abot.Settings.view = function(ctrl) {
	return m(".body", [
		m.component(abot.Header),
		m(".container", [
			m.component(abot.Sidebar, { active: 5 }),
			m(".main", [
				m(".topbar", "Settings"),
				m(".content", [
					function() {
						if (ctrl.props.error().length > 0) {
							return m(".alert.alert-danger.alert-margin", ctrl.props.error())
						}
						if (ctrl.props.success().length > 0) {
							return m(".alert.alert-success.alert-margin", ctrl.props.success())
						}
					}(),
					function() {
						if (ctrl.props.plugins().length === 0) {
							return m("p.top-el", "No plugins installed.")
						}
						return ctrl.props.plugins().map(function(plugin, i) {
							var title
							if (i === 0) {
								title = m("h3.top-el", plugin.Name)
							} else {
								title = m("h3", plugin.Name)
							}
							var ss = []
							Object.keys(plugin.Settings).map(function(setting) {
								ss.push(m("tr", [
									m("td", [
										m("label", setting),
									]),
									m("td", [
										m("input[type=text]", {
											oninput: m.withAttr("value", ctrl.markChanged),
											"data-plugin-idx": i,
											"data-setting": setting,
											value: plugin.Settings[setting],
										}),
									]),
								]))
							})
							return m("form.form-align", [
								title,
								m("table", [ ss ]),
							])
						})
					}(),
					m("#btns.btn-container-left", [
						m("input[type=button].btn", {
							onclick: ctrl.discard,
							value: "Discard Changes",
						}),
						m("input[type=button].btn.btn-primary", {
							onclick: ctrl.save,
							value: "Save",
						}),
					]),
				]),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
