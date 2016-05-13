(function(abot) {
abot.Index = {}
abot.Index.controller = function() {
	if (abot.isAdmin()) {
		return m.route("/admin", null, true)
	}
	if (abot.isLoggedIn()) {
		return m.route("/profile", null, true)
	}
	var ctrl = this
	ctrl.submit = function(ev) {
		ev.preventDefault()
		var name = document.getElementById("name").value
		var email = document.getElementById("email").value
		var pass = document.getElementById("password").value
		var flexId = document.getElementById("phone").value
		return m.request({
			method: "POST",
			data: {
				Name: name,
				Email: email,
				Password: pass,
				FID: flexId,
				Admin: true,
			},
			url: "/api/signup.json"
		}).then(function(data) {
			var date = new Date()
			var exp = date.setDate(date + 30)
			var secure = abot.isProduction()
			Cookies.set("id", data.ID, exp, null, null, secure)
			Cookies.set("email", data.Email, exp, null, null, secure)
			Cookies.set("issuedAt", data.IssuedAt, exp, null, null, secure)
			Cookies.set("authToken", data.AuthToken, exp, null, null, secure)
			Cookies.set("csrfToken", data.CSRFToken, exp, null, null, secure)
			Cookies.set("scopes", data.Scopes, exp, null, null, secure)
			m.route("/profile")
		}, function(err) {
			ctrl.props.error(err.Msg)
		})
	}
	m.request({
		url: "/api/admin_exists.json",
		method: "GET",
	}).then(function(resp) {
		if (resp === true) {
			return m.route("/login", null, true)
		}
		abot.Index.view = abot.Index.loadedView
	}, function(err) {
		console.log(err.Msg)
	})
}
abot.Index.view = function() {
	return m(".container", [
		m.component(abot.Header),
	])
}
abot.Index.loadedView = function(ctrl) {
	return m(".container", [
		m.component(abot.Header),
		m(".main-no-sidebar", [
			m(".centered", [
				m("img[src=/public/images/logo.svg].big-icon"),
			]),
			m(".well.well-form", [
				m(".centered", [ m("h3.top-el", "Welcome to Abot!") ]),
				m("p.top-el", [
					"Before you start building your bot, let's set up an admin account.",
				]),
				m("form.centered", { onsubmit: ctrl.submit }, [
					m(".form-el", m("input#name[type=text][placeholder=Name]", {
						config: this.focus,
					})),
					m(".form-el", m("input#email[type=email]", {
						placeholder: "Email",
					})),
					m(".form-el", m("input#phone[type=tel]", {
						placeholder: "Phone number",
					})),
					m(".form-el", m("input#password[type=password]]", {
						placeholder: "Password",
					})),
					m(".form-el", m("input[type=submit].btn", {
						value: "Create Admin Account",
					})),
				]),
			]),
		])
	])
}
})(!window.abot ? window.abot={} : window.abot);
