var ResetPassword = {
	submit: function(ev) {
		ev.preventDefault();
		ResetPassword.vm.hideError();
		var password = document.getElementById("pw").value;
		return m.request({
			method: "POST",
			data: {
				Secret: m.route.param("s"),
				Password: password
			},
			url: "/api/reset_password.json"
		}).then(function(data) {
			ResetPassword.vm.showSuccess();
		}, function(err) {
			ResetPassword.vm.showError(err.Msg);
		});
	},
	checkAuth: function(callback) {
		if (cookie.getItem("id") !== null) {
			callback(true);
		}
	}
};

ResetPassword.controller = function() {
	ResetPassword.checkAuth(function(loggedIn) {
		if (loggedIn) {
			return m.route("/profile");
		}
	});
	ResetPassword.controller.error = m.prop("");
	ResetPassword.controller.success = m.prop("");
};

ResetPassword.vm = {
	hideError: function() {
		ResetPassword.controller.error("");
		document.getElementById("err").classList.add("hidden");
	},
	showError: function(err) {
		ResetPassword.controller.error(err);
		document.getElementById("err").classList.remove("hidden");
	},
	showSuccess: function() {
		ResetPassword.controller.success("Successfully reset your password.")
		document.getElementById("success").classList.remove("hidden");
		document.getElementById("form").classList.add("hidden");
		document.getElementById("btn").classList.add("hidden");
	}
};

ResetPassword.view = function() {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		ResetPassword.viewFull(),
		Footer.view()
	]);
}

ResetPassword.viewFull = function() {
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
							}, ResetPassword.controller.error()),
							m("div", {
								id: "success",
								class: "alert alert-success hidden"
							}, ResetPassword.controller.success()),
							m("div", {
								id: "form"
							}, [
								m("p", "Please set your new password below."),
								m("div", {
									class: "form-group"
								}, [
									m("input", {
										type: "password",
										class: "form-control",
										id: "pw",
										placeholder: "New password"
									}),
								]),
								m("div", {
									class: "form-group"
								}, [
									m("input", {
										type: "password",
										class: "form-control",
										id: "pw2",
										placeholder: "Confirm new password"
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
									onclick: ResetPassword.submit,
									onsubmit: ResetPassword.submit,
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
							m("a", {
								href: "/login",
								config: m.route
							}, "Login here")
						])
					])
				])
			])
		])
	]);
};
