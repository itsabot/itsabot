(function(abot) {
abot.Login = {}
abot.Login.controller = function() {
	var ctrl = this
	ctrl.login = function(ev) {
		ev.preventDefault()
		ctrl.hideError()
		var email = document.getElementById("email").value
		var pass = document.getElementById("password").value
		return m.request({
			method: "POST",
			data: {
				email: email,
				password: pass
			},
			url: "/api/login.json"
		}).then(function(data) {
			var date = new Date()
			var exp = date.setDate(date + 30)
			var secure = true
			if (window.location.hostname === "localhost") {
				secure = false
			}
			cookie.setItem("id", data.Id, exp, null, null, secure)
			cookie.setItem("trainer", data.Trainer, exp, null, null, secure)
			cookie.setItem("session_token", data.SessionToken, exp, null, null, secure)
			if (m.route.param("r") == null) {
				return m.route("/profile")
			}
			m.route(decodeURIComponent(m.route.param("r")).substr(1))
		}, function(err) {
			ctrl.showError(err.Msg)
		})
	}
	abot.Login.checkAuth(function(loggedIn) {
		if (loggedIn) {
			return m.route("/profile")
		}
	})
	ctrl.hideError = function() {
		ctrl.error("")
		document.getElementById("err").classList.add("hidden")
	}
	ctrl.showError = function(err) {
		ctrl.error(err)
		document.getElementById("err").classList.remove("hidden")
	}
	ctrl.error = m.prop("")
}
abot.Login.view = function(ctrl) {
	return m(".main", [
		m.component(abot.Header),
		m("#full.container", m(".row.margin-top-sm", m(".col-md-push-3.col-md-6.card", [
			m(".row", [
				m(".col-md-12.text-center", [
					m("h2", "Log In")
				])
			]),
			m("form", [
				m(".row.margin-top-sm", [
					m(".col-md-12", [
						m("div", {
							id: "err",
							class: "alert alert-danger hidden"
						}, ctrl.error()),
						m(".form-group", [
							m("input", {
								type: "email",
								class: "form-control",
								id: "email",
								placeholder: "Email"
							})
						]),
						m(".form-group", [
							m("input", {
								type: "password",
								class: "form-control",
								id: "password",
								placeholder: "Password"
							})
						]),
						m(".form-group.text-right", [
							m("a", {
								href: "/forgot_password",
								config: m.route
							}, "Forgot password?")
						])
					])
				]),
				m(".row", [
					m(".col-md-12.text-center", [
						m(".form-group", [
							m("input", {
								class: "btn btn-sm",
								id: "btn",
								type: "submit",
								onclick: ctrl.login,
								onsubmit: ctrl.login,
								value: "Log In"
							})
						])
					])
				])
			]),
			m(".row", [
				m(".col-md-12.text-center", [
					m(".form-group", [
						m("span", "No account? "),
						m("a", {
							href: "/signup",
							config: m.route
						}, "Sign Up")
					])
				])
			])
		])))
	])
}
abot.Login.checkAuth = function(callback) {
	if (cookie.getItem("id") !== null) {
		callback(true)
	}
}
})(!window.abot ? window.abot={} : window.abot);
