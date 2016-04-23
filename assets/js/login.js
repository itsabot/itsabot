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
				password: pass,
			},
			url: "/api/login.json",
		}).then(function(data) {
			var date = new Date()
			var exp = date.setDate(date + 30)
			var secure = abot.isProduction()
			if (window.location.hostname === "localhost") {
				secure = false
			}
			console.log(data)
			Cookies.set("id", data.ID, exp, null, null, secure)
			Cookies.set("email", data.Email, exp, null, null, secure)
			Cookies.set("issuedAt", data.IssuedAt, exp, null, null, secure)
			Cookies.set("authToken", data.AuthToken, exp, null, null, secure)
			Cookies.set("csrfToken", data.CSRFToken, exp, null, null, secure)
			Cookies.set("scopes", data.Scopes, exp, null, null, secure)
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
	return m(".container", [
		m.component(abot.Header),
		m(".main-no-sidebar", [
			m(".centered.content", [
				m("h1", "Log In"),
				m(".well.well-form", [
					m("div", {
						id: "err",
						"class": "alert alert-danger alert-margin hidden",
					}, ctrl.error()),
					m("form", { onsubmit: ctrl.login }, [
						m(".form-el", [
							m("input#email[type=email]", {
								placeholder: "Email"
							}),
						]),
						m(".form-el", [
							m("input#password[type=password]", {
								placeholder: "Password"
							}),
						]),
						m("small", [
							m("a[href=/forgot_password]", {
								config: m.route,
							}, "Forgot password?")
						]),
						m(".form-el", [
							m("input.btn#btn[type=submit]", { value: "Log In" }),
						]),
					]),
					m("small", [
						m("span", "No account? "),
						m("a[href=/signup]", { config: m.route }, "Sign Up"),
					]),
				]),
			]),
		]),
	])
}
abot.Login.checkAuth = function(callback) {
	var id = Cookies.get("id")
	var issuedAt = Cookies.get("issuedAt")
	var email = Cookies.get("email")
	if (id != null && id !== "undefined" &&
		issuedAt != null && issuedAt !== "undefined" &&
		email != null && email !== "undefined") {
		return callback(true)
	}
	return callback(false)
}
})(!window.abot ? window.abot={} : window.abot);
