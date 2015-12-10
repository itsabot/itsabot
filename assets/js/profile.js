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
	},
	sendView: function(uid) {
		return m.request({
			method: "PUT",
			url: "/api/profile.json",
			data: { UserID: parseInt(uid, 10) }
		});
	}
};

Profile.controller = function() {
	var userId = cookie.getItem("id");
	console.log(userId);
	if (userId === null || userId <= 0) {
		cookie.removeItem("id");
		cookie.removeItem("session_token");
		return m.route("/login");
	}
	var redirect = m.route.param("r");
	console.log(redirect);
	if (redirect != null) {
		m.route("/" + redirect.substring(1));
	}
	var _this = this;
	_this.username = m.prop("");
	_this.email = m.prop("");
	_this.phones = new List({type: Phone});
	_this.cards = new List({type: Card});
	Profile.data(userId).then(function(data) {
		_this.email(data.Email);
		_this.username(data.Name);
		_this.phones.userId(userId);
		if (data.Phones === null) {
			data.Phones = [];
		}
		_this.phones.data(data.Phones);
		_this.cards.userId(userId);
		if (data.Cards === null) {
			data.Cards = [];
		}
		var cards = [];
		for (var i = 0; i < data.Cards.length; ++i) {
			var card = new Card();
			var c = data.Cards[i];
			card.id(c.Id);
			card.cardholderName(c.CardholderName);
			card.brand(c.Brand);
			card.expMonth(c.ExpMonth);
			card.expYear(c.ExpYear);
			card.last4(c.Last4);
			cards.push(card);
		}
		_this.cards.data(cards);
	}, function(err) {
		console.error(err);
	});
	Profile.sendView(userId);
};

Profile.view = function(controller) {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Profile.viewFull(controller),
		Footer.view()
	]);
};

Profile.viewFull = function(controller) {
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
								m("div", controller.email())
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
									value: controller.username()
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
							controller.phones.view()
						])
					]),
					m("h3", {
						class: "margin-top-sm"
					}, "Credit cards"),
					m("div", {
						class: "form-group card"
					}, [
						m("div", [
							controller.cards.view(),
							m("div", [
								m("a", {
									id: controller.cards.id + "-add-btn",
									class: "btn btn-sm",
									href: "/cards/new",
									config: m.route
								}, "+Add Card")
							])
						])
					])
				])
			])
		])
	]);
};
