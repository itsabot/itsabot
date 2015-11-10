var Signup = {
	signup: function(ev) {
		ev.preventDefault();
		var name = document.getElementById("name").value;
		var email = document.getElementById("email").value;
		var pass = document.getElementById("password").value;
		var flexId = document.getElementById("phone").value;
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
			var date = new Date();
			var exp = date.setDate(date + 30);
			cookie.setItem("id", data.Id, exp, null, null, false);
			cookie.setItem("customer_id", data.CustomerId, exp, null, null, false);
			cookie.setItem("session_token", data.SessionToken, exp, null, null, false);
			m.route("/profile");
		}, function(err) {
			Signup.controller.error(err.Msg);
		});
	}
};

Signup.controller = function() {
	Login.checkAuth(function(cb) {
		if (cb) {
			return m.route("/profile");
		}
	});
	var name = m.route.param("name") || "";
	var phone = m.route.param("fid") || "";
	Signup.controller.userName = m.prop(name);
	Signup.controller.phone = m.prop(phone);
	Signup.controller.error = m.prop("");
};

Signup.vm = {
	phoneDisabled: function() {
		return Signup.controller.phone().length > 0;
	}
};

Signup.view = function() {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Signup.viewFull(),
		Footer.view()
	]);
};

Signup.viewFull = function() {
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
						m("h2", "Sign Up")
					])
				]),
				m("form", {
					onsubmit: Signup.signup
				}, [
					m("div", {
						class: "row margin-top-sm"
					}, [
						m("div", {
							class: "col-md-12"
						}, [

							function() {
								if (Signup.controller.error() !== "") {
									return m("div", {
										class: "alert alert-danger"
									}, Signup.controller.error());
								}
							}(),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "text",
									class: "form-control",
									id: "name",
									placeholder: "Your name"
								})
							]),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "tel",
									class: "form-control",
									id: "phone",
									placeholder: "Your phone number",
									value: Signup.controller.phone(),
									disabled: Signup.vm.phoneDisabled()
								})
							]),
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
									value: "Sign Up"
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
							m("span", "Have an account? "),
							m("a", {
								href: "/login",
								config: m.route
							}, "Log In")
						])
					])
				])
			])
		])
	]);
};
