(function(abot) {
abot.Profile = {}
abot.Profile.controller = function() {
	var userId = Cookies.get("id")
	if (!abot.isLoggedIn()) {
		return m.route("/login")
	}
	var redirect = m.route.param("r")
	if (!!redirect) {
		m.route("/" + redirect.substring(1))
	}
	var ctrl = this
	ctrl.data = function(uid) {
		return abot.request({
			method: "GET",
			url: "/api/user/profile.json",
		})
	},
	ctrl.sendView = function() {
		return abot.request({
			method: "PUT",
			url: "/api/user/profile.json",
		})
	}
	ctrl.props = {
		username: m.prop(""),
		email: m.prop(""),
		phones: m.prop([]),
	}
	ctrl.data(userId).then(function(data) {
		ctrl.props.email(data.Email)
		ctrl.props.username(data.Name)
		ctrl.props.phones(data.Phones || [])
	}, function(err) {
		console.error(err)
	})
	ctrl.sendView(userId)
}
abot.Profile.view = function(ctrl) {
	return m(".container", [
		m.component(abot.Header),
		m.component(abot.Sidebar, { active: -1 }),
		m(".main", [
			m(".topbar", "Profile"),
			m(".content", [
				m("h3.top-el", "Account Details"),
				m("div", [
					m("label", "Username"),
					m("div", m("div", ctrl.props.email())),
				]),
				m(".form-el", [
					m("label", "Password"),
					m("div", m("a[href=#]", "Change password")),
				]),
				m(".form-el", [
					m("label", { for: "username" }, "Name"),
					m("div", [
						m("input#username[type=text]", {
							value: ctrl.props.username(),
						}),
					]),
				]),
				m.component(abot.Phones, ctrl.props.phones()),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
