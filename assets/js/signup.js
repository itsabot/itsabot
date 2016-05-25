(function(abot) {
abot.Signup = {}
abot.Signup.controller = function() {
	if (abot.isLoggedIn()) {
		return m.route("/profile", null, true)
	}
	var ctrl = this
	ctrl.signup = function(ev) {
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
	ctrl.focus = function(el) {
		el.focus()
	}
	ctrl.phoneDisabled = function() {
		return ctrl.props.phone().length > 0
	}
	ctrl.props = {
		userName: m.prop(m.route.param("name") || ""),
		phone: m.prop(m.route.param("fid") || ""),
		error: m.prop("")
	}
}
abot.Signup.view = function(ctrl) {
	var errMsg = null
	if (!!ctrl.props.error()) {
		errMsg = m(".alert.alert-danger", ctrl.props.error())
	}
	return m(".container", [
		m.component(abot.Header),
		m(".main-no-sidebar", [
			m(".centered.content", [
				m("h1", "Sign Up"),
				m(".well.well-form", [
					m(".well-padding", [
						errMsg,
						m("form", { onsubmit: ctrl.signup }, [
							m(".form-el", [
								m("input", {
									type: "text",
									class: "form-control",
									id: "name",
									placeholder: "Your name",
									config: ctrl.focus,
								})
							]),
							m(".form-el", [
								m("input", {
									type: "tel",
									class: "form-control",
									id: "phone",
									placeholder: "Your phone number",
									value: ctrl.props.phone(),
									disabled: ctrl.phoneDisabled()
								})
							]),
							m(".form-el", [
								m("input", {
									type: "email",
									class: "form-control",
									id: "email",
									placeholder: "Email",
								})
							]),
							m(".form-el", [
								m("input", {
									type: "password",
									class: "form-control",
									id: "password",
									placeholder: "Password"
								})
							]),
							m(".form-el", [
								m("input", {
									class: "btn btn-sm",
									id: "btn",
									type: "submit",
									value: "Sign Up"
								})
							]),
							m("small", [
								m("span", "Have an account? "),
								m("a", {
									href: "/login",
									config: m.route
								}, "Log In")
							]),
						]),
					]),
				]),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
