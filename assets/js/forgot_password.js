(function(abot) {
abot.ForgotPassword = {}
abot.ForgotPassword.controller = function() {
	var ctrl = this
	ctrl.submit = function(ev) {
		ev.preventDefault()
		console.log("preventing default")
		ctrl.hideError()
		var email = document.getElementById("email").value
		return abot.request({
			method: "POST",
			data: { Email: email },
			url: "/api/forgot_password.json"
		}).then(function(data) {
			ctrl.showSuccess()
		}, function(err) {
			console.log("Error!")
			ctrl.showError(err.Msg)
		})
	},
	ctrl.checkAuth = function(callback) {
		if (Cookies.get("id") !== null) {
			callback(true)
		}
	}
	ctrl.checkAuth(function(loggedIn) {
		if (loggedIn) {
			return m.route("/profile")
		}
	})
	ctrl.error = m.prop("")
	ctrl.success = m.prop("")
	ctrl.hideError = function() {
		ctrl.error("")
		document.getElementById("err").classList.add("hidden")
	},
	ctrl.showError = function(err) {
		ctrl.error(err)
		document.getElementById("err").classList.remove("hidden")
	},
	ctrl.showSuccess = function() {
		ctrl.success("We've emailed you a link to reset your password. Please open that link to continue. For security reasons the link will expire in 30 minutes.")
		document.getElementById("success").classList.remove("hidden")
		document.getElementById("form").classList.add("hidden")
		document.getElementById("btn").classList.add("hidden")
	}
}
abot.ForgotPassword.view = function(ctrl) {
	return m(".main", [
		m.component(abot.Header),
		abot.ForgotPassword.viewFull(ctrl),
	])
}
abot.ForgotPassword.viewFull = function(ctrl) {
	return m("#full.container", [
		m(".row.margin-top-sm", [
			m(".col-md-push-3.col-md-6.card", [
				m(".row", [
					m(".col-md-12.text-center", [
						m("h2", "Reset Password")
					])
				]),
				m("form", [
					m(".row.margin-top-sm", [
						m(".col-md-12", [
							m("div", {
								id: "err",
								class: "alert alert-danger hidden"
							}, ctrl.error()),
							m("div", {
								id: "success",
								class: "alert alert-success hidden"
							}, ctrl.success()),
							m("#form", [
								m("p", "We'll email you a confirmation link to reset your password. Please enter your email below."),
								m(".form-group", [
									m("input", {
										type: "email",
										class: "form-control",
										id: "email",
										placeholder: "Email"
									})
								])
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
									onclick: ctrl.submit,
									value: "Submit"
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
			])
		])
	])
}
})(!window.abot ? window.abot={} : window.abot);
