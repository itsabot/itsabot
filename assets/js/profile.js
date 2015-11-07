var Profile = {
	signout: function(ev) {
		ev.preventDefault();
		cookie.removeItem("id");
		cookie.removeItem("session_token");
		m.route("/login");
	},
	data: function(uid) {
		return m.request({
			method: "GET",
			url: "/api/profile.json?uid=" + uid
		});
	}
};
Profile.vm = function() {
	var userId = cookie.getItem("id");
	Profile.data(userId).then(function(data) {
		Profile.controller.email(data.Email);
		Profile.controller.username(data.Name);
		Profile.controller.phoneList.userId(userId);
		Profile.controller.phoneList.data(data.Phones);
		Profile.controller.phoneList.showAdd(false);
		Profile.controller.phoneList.type("phones");
		Profile.controller.cards.userId(userId);
		Profile.controller.cards.data(data.Cards);
		Profile.controller.cards.showAdd(true);
		Profile.controller.cards.type("cards");
	}, function(err) {
		console.log(err);
	});
};
Profile.controller = function() {
	console.log(Profile.controller.name);
	if (cookie.getItem("id") === null) {
		return m.route("/login");
	}
	Profile.controller.username = m.prop("");
	Profile.controller.email = m.prop("");
	Profile.controller.phoneList = new List();
	Profile.controller.cards = new List();
	Profile.vm();
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
				class: "col-md-12"
			}, [
				m("h1", "Profile")
			])
		]),
		m("div", {
			class: "row"
		}, [
			m("div", {
				class: "col-md-7 margin-top-sm"
			}, [
				m("h3", "Account Details"),
				m("form", {
					class: "margin-top-sm"
				}, [
					m("div", {
						class: "card"
					}, [
						m("div", {
							class: "form-group"
						}, [
							m("label", "Username"),
							m("div", [
								m("div", Profile.controller.email())
							])
						]),
						m("div", {
							class: "form-group"
						}, [
							m("label", "Password"),
							m("div", [
								m("a", {
									href: "#"
								}, "Change password")
							])
						]),
						m("div", {
							class: "form-group"
						}, [
							m("label", {
								for: "username"
							}, "Name"),
							m("div", [
								m("input", {
									id: "username",
									type: "text",
									class: "form-control",
									value: Profile.controller.username()
								})
							])
						]),
						m("div", {
							class: "form-group margin-top-sm"
						}, [
							m("div", [
								m("a", {
									class: "btn btn-sm",
									href: "#/",
									onclick: Profile.signout
								}, "Sign out")
							])
						])
					]),
					m("h3", {
						class: "margin-top-sm"
					}, "Phone numbers"),
					m("div", {
						class: "form-group card"
					}, [
						m("div", [
							Profile.controller.phoneList.view()
						])
					]),
					m("h3", {
						class: "margin-top-sm"
					}, "Credit cards"),
					m("div", {
						class: "form-group card"
					}, [
						m("div", [
							Profile.controller.cards.view()
						])
					])
				])
			])
		])
	]);
};
