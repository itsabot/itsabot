var Profile = {};
Profile.controller = function() {
	var userId = cookie.getItem("id");
	if (userId === null) {
		return m.route("/login");
	}
	Profile.controller.phoneList = new List();
	Profile.controller.phoneList.type("phones");
	Profile.controller.phoneList.userId(userId);
	Profile.controller.phoneList.showAdd(false);
	Profile.controller.phoneList.data().then(function(phones) {
		Profile.controller.phones = m.prop(phones);
	}, function(err) {
		if (err !== null) {
			console.log(err);
		}
	});
};
Profile.view = function() {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Profile.viewFull(),
		Footer.view()
	]);
};

Profile.viewFull = function() {
	return m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-sm-12"
			}, [
				m("h1", "Profile")
			])
		]),
		m("div", {
			class: "row"
		}, [
			m("div", {
				class: "col-sm-12 margin-top-sm"
			}, [
				m("h2", "Account Details"),
				m("form", {
					class: "form-horizontal margin-top-sm"
				}, [
					m("div", {
						class: "form-group"
					}, [
						m("label", {
							for: "loginEmail",
							class: "col-md-2",
							value: "egtann"
						}, "Login"),
						m("div", {
							class: "col-md-5"
						}, [
							m("input", {
								id: "loginEmail",
								type: "text",
								class: "form-control",
								readonly: "true"
							})
						])
					]),
					m("div", {
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-2"
						}, "Password"),
						m("div", {
							class: "col-md-5"
						}, [
							m("a", {
								href: "#"
							}, "Change password")
						])
					]),
					m("div", {
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-2",
							for: "phoneNums"
						}, "Phone numbers"),
						m("div", {
							class: "col-md-5"
						}, [
							Profile.controller.phoneList.view(Profile.controller.phones())
						])
					])
				])
			])
		])
	]);
};
