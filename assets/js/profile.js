(function(abot) {
abot.Profile = {}
abot.Profile.controller = function() {
	var userId = cookie.getItem("id")
	if (!userId || userId <= 0) {
		cookie.removeItem("id")
		cookie.removeItem("trainer")
		cookie.removeItem("session_token")
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
	return m(".main", [
		m.component(abot.Header),
		abot.Profile.viewFull(ctrl),
	])
}
abot.Profile.viewFull = function(ctrl) {
	console.log("hit");
	return m(".profile", [
		m("h1", "Profile"),
		m("h2", "Account Details"),
		m("label", "Username"),
		m("div", m("div", ctrl.props.email())),
		m("label", "Password"),
		m("div", m("a[href=#]", "Change password")),
		m("label", {
			for: "username"
		}, "Name"),
		m("input#username[type=text]", {
			value: ctrl.props.username(),
		}),
		m.component(abot.Phones, ctrl.props.phones()),
	])
}
})(!window.abot ? window.abot={} : window.abot);
