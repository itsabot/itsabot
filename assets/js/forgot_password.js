var ForgotPassword = {
	submit: function(ev) {
		ev.preventDefault();
		ForgotPassword.vm.hideError();
		var email = document.getElementById("email").value;
		return m.request({
			method: "POST",
			data: {
				Email: email
			},
			url: "/api/forgot_password.json"
		}).then(function(data) {
			ForgotPassword.vm.showSuccess();
		}, function(err) {
			ForgotPassword.vm.showError(err.Msg);
		});
	},
	checkAuth: function(callback) {
		if (cookie.getItem("id") !== null) {
			callback(true);
		}
	}
};

ForgotPassword.controller = function() {
	ForgotPassword.checkAuth(function(loggedIn) {
		if (loggedIn) {
			return m.route("/profile");
		}
	});
	ForgotPassword.controller.error = m.prop("");
	ForgotPassword.controller.success = m.prop("");
};

ForgotPassword.vm = {
	hideError: function() {
		ForgotPassword.controller.error("");
		document.getElementById("err").classList.add("hidden");
	},
	showError: function(err) {
		ForgotPassword.controller.error(err);
		document.getElementById("err").classList.remove("hidden");
	},
	showSuccess: function() {
		ForgotPassword.controller.success("We've emailed you a link to reset your password. Please open that link to continue. For security reasons the link will expire in 30 minutes.");
		document.getElementById("success").classList.remove("hidden");
		document.getElementById("form").classList.add("hidden");
		document.getElementById("btn").classList.add("hidden");
	}
};

ForgotPassword.view = function() {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		ForgotPassword.viewFull(),
		Footer.view()
	]);
}

ForgotPassword.viewFull = function() {
	return m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-md-push-3 col-md-6 card"
			}, [
				m("div", {
					class: "row"
				}, [
					m("div", {
						class: "col-md-12 text-center"
					}, [
						m("h2", "Reset Password")
					])
				]),
				m("form", [
					m("div", {
						class: "row margin-top-sm"
					}, [
						m("div", {
							class: "col-md-12"
						}, [
							m("div", {
								id: "err",
								class: "alert alert-danger hidden"
							}, ForgotPassword.controller.error()),
							m("div", {
								id: "success",
								class: "alert alert-success hidden"
							}, ForgotPassword.controller.success()),
							m("div", {
								id: "form"
							}, [
								m("p", "We'll email you a confirmation link to reset your password. Please enter your email below."),
								m("div", {
									class: "form-group"
								}, [
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
					m("div", {
						class: "row"
					}, [
						m("div", {
							class: "col-md-12 text-center"
						}, [
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									class: "btn btn-sm",
									id: "btn",
									type: "submit",
									onclick: ForgotPassword.submit,
									onsubmit: ForgotPassword.submit,
									value: "Submit"
								})
							])
						])
					])
				]),
				m("div", {
					class: "row"
				}, [
					m("div", {
						class: "col-md-12 text-center"
					}, [
						m("div", {
							class: "form-group"
						}, [
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
	]);
};
