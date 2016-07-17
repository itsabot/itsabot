(function(abot) {
abot.ManageTeam = {}
abot.ManageTeam.controller = function() {
	var ctrl = this
	ctrl.props = {
		admins: m.prop([]),
		error: m.prop(""),
		success: m.prop(""),
	}
	ctrl.addAdmin = function(ev) {
		ev.preventDefault()
		var v = document.getElementById("admin-email").value
		if (v.length === 0) {
			return
		}
		abot.request({
			url: "/api/admins.json",
			method: "PUT",
			data: {
				Email: v,
				Admin: true,
			},
		}).then(function(resp) {
			document.getElementById("admin-email").value = ""
			ctrl.props.error("")
			ctrl.props.success("Success! Added admin " + v)
			ctrl.props.admins().push(resp)
		}, function(err) {
			ctrl.props.success("")
			ctrl.props.error(err.Msg)
		})
	}
	abot.request({
		url: "/api/admins.json",
		method: "GET",
	}).then(ctrl.props.admins, function(err) {
		ctrl.props.error(err.Msg)
	})
}
abot.ManageTeam.view = function(ctrl) {
	return m(".body", [
		m.component(abot.Header),
		m(".container", [
			m.component(abot.Sidebar, { active: 3 }),
			m(".main", [
				m(".topbar", "Manage Team"),
				m(".content", [
					m("h3.top-el", "Admins"),
					function() {
						if (ctrl.props.error().length > 0) {
							return m(".alert.alert-danger.alert-margin", ctrl.props.error())
						}
						if (ctrl.props.success().length > 0) {
							return m(".alert.alert-success.alert-margin", ctrl.props.success())
						}
					}(),
					m("p", "Removing permissions will take effect immediately. Adding permissions will require that user to log out and back in again to be available."),
					m("table.table-compact", [
						m("thead", [
							m("th", ""),
							m("th", "Name"),
							m("th", "Email"),
						]),
						function() {
							var c = []
							var a = ctrl.props.admins()
							var e = Cookies.get("email")
							// Store this bool so we can stop doing expensive
							// string comparisons once we've found the user
							var looking = true
							for (var i = 0; i < a.length; ++i) {
								if (looking || a[i].Email === e) {
									c.unshift(m.component(abot.TableItemUser, ctrl, a[i]))
									looking = false
								} else {
									c.push(m.component(abot.TableItemUser, ctrl, a[i]))
								}
							}
							return c
						}(),
					]),
					m(".well.well-no-border", [
						m("form", { onsubmit: ctrl.addAdmin }, [
							m("h4.form-header", "Add an admin"),
							m("input#admin-email[type=email]", { placeholder: "Email" }),
							m("input.btn.btn-inline[type=submit]", { value: "Add admin" }),
						]),
					]),
				]),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
