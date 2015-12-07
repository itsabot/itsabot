var Login = {
	login: function(ev) {
		ev.preventDefault();
		Login.vm.hideError();
		var email = document.getElementById("email").value;
		var pass = document.getElementById("password").value;
		return m.request({
			method: "POST",
			data: {
				email: email,
				password: pass
			},
			url: "/api/login.json"
		}).then(function(data) {
			var date = new Date();
			var exp = date.setDate(date + 30);
			var secure = true;
			if (window.location.hostname === "localhost") {
				secure = false;
			}
			cookie.setItem("id", data.Id, exp, null, null, secure);
			cookie.setItem("session_token", data.SessionToken, exp, null, null, secure);
			var redirect = m.route.param("r");
			if (redirect != null) {
				m.route(redirect.substring(1));
			} else {
				m.route("/profile");
			}
		}, function(err) {
			Login.vm.showError(err.Msg);
		});
	},
	checkAuth: function(callback) {
		if (cookie.getItem("id") !== null) {
			callback(true);
		}
	}
};

Login.controller = function() {
	Login.checkAuth(function(loggedIn) {
		if (loggedIn) {
			return m.route("/profile");
		}
	});
	Login.controller.error = m.prop("");
};

Login.vm = {
	hideError: function() {
		Login.controller.error("");
		document.getElementById("err").classList.add("hidden");
	},
	showError: function(err) {
		Login.controller.error(err);
		document.getElementById("err").classList.remove("hidden");
	}
};

Login.view = function() {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Login.viewFull(),
		Footer.view()
	]);
}

Login.viewFull = function() {
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
						m("h2", "Log In")
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
							}, Login.controller.error()),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "email",
									class: "form-control",
									id: "email",
									placeholder: "Email"
								})
							]),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "password",
									class: "form-control",
									id: "password",
									placeholder: "Password"
								})
							]),
							m("div", {
								class: "form-group text-right"
							}, [
								m("a", "Forgot password?")
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
									onclick: Login.login,
									onsubmit: Login.login,
									value: "Log In"
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
