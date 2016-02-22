(function(abot) {
abot.Signup = {}
abot.Signup.controller = function() {
	var ctrl = this
	abot.Login.checkAuth(function(cb) {
		if (cb) {
			return m.route("/profile")
		}
	})
	ctrl.props = {
		userName: m.prop(m.route.param("name") || ""),
		phone: m.prop(m.route.param("fid") || ""),
		error: m.prop("")
	}
	ctrl.signup = function(ev) {
		ev.preventDefault()
		var name = document.getElementById("name").value
		var email = document.getElementById("email").value
		var pass = document.getElementById("password").value
		var flexId = document.getElementById("phone").value
		return m.request({
			method: "POST",
			data: {
				name: name,
				email: email,
				password: pass,
				fid: flexId
			},
			url: "/api/signup.json"
		}).then(function(data) {
			var date = new Date()
			var exp = date.setDate(date + 30)
			cookie.setItem("id", data.Id, exp, null, null, false)
			cookie.setItem("session_token", data.SessionToken, exp, null, null, false)
			m.route("/profile")
		}, function(err) {
			ctrl.props.error(err.Msg)
		})
	}
	ctrl.phoneDisabled = function() {
		return ctrl.props.phone().length > 0
	}
}
abot.Signup.view = function(ctrl) {
	var errMsg = null
	if (!!ctrl.props.error()) {
		errMsg = m(".alert.alert-danger", ctrl.props.error())
	}
	return m(".main", [
		m.component(abot.Header),
		m("#full.container", m(".row.margin-top-sm", [
			m(".col-md-push-3.col-md-6.card", [
				m(".row", [
					m(".col-md-12.text-center", [
						m("h2", "Sign Up")
					])
				]),
				m("form", { onsubmit: ctrl.signup }, [
					m(".row.margin-top-sm", [
						m(".col-md-12", [
							errMsg,
							m(".form-group", [
								m("input", {
									type: "text",
									class: "form-control",
									id: "name",
									placeholder: "Your name"
								})
							]),
							m(".form-group", [
								m("input", {
									type: "tel",
									class: "form-control",
									id: "phone",
									placeholder: "Your phone number",
									value: ctrl.props.phone(),
									disabled: ctrl.phoneDisabled()
								})
							]),
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
									value: "Sign Up"
								})
							])
						])
					])
				]),
				m(".row", [
					m(".col-md-12.text-center", [
						m(".form-group", [
							m("span", "Have an account? "),
							m("a", {
								href: "/login",
								config: m.route
							}, "Log In")
						])
					])
				])
			])
		]))
	])
}
})(!window.abot ? window.abot={} : window.abot);
