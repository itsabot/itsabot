(function(ava) {
ava.TrainShow = {}
ava.TrainShow.controller = function() {
	if (!ava.isLoggedIn()) {
		m.route("/login?r=" + encodeURIComponent(window.location.search))
		return
	}
	var ctrl = this
	id = m.route.param("id")
	uid = m.route.param("uid")
	ctrl.loadConversation = function() {
		m.request({
			method: "GET",
			url: "/api/conversation.json?id=" + id + "&uid=" + uid
		}).then(function(resp) {
			ctrl.props.Messages(resp.Chats)
			ctrl.props.Username(resp.Username)
		})
	}
	ctrl.toggleCalSelector = function() {
		document.getElementById("calendar-selector").classList.toggle("hidden")
	},
	ctrl.toggleUserPrefs = function() {
		document.getElementById("user-prefs").classList.toggle("hidden")
	},
	ctrl.toggleContactSearch = function() {
		document.getElementById("contact-search").classList.toggle("hidden")
	},
	ctrl.toggleAccountInfo = function() {
		document.getElementById("edit-account-info").classList.toggle("hidden")
	}
	ctrl.newChatWindow = function() {
		ctrl.props.chatWindowsOpen(ctrl.props.chatWindowsOpen()+1)
	}
	ctrl.confirmComplete = function() {
		if (confirm("Are you sure you want to mark the conversation as complete? You won't be able to message the user again.")) {
			var userId = cookie.getItem("id")
			m.request({
				method: "PATCH",
				url: "/api/conversation.json?uid=" + userId,
			}).then(function() {
				m.route("/train?trained=true")
			}, function(err) {
				console.error(err)
			})
		}
	}
	ctrl.confirmAddCalendar = function() {
		if (confirm("Are you sure you want to have the user add a calendar?")) {
			m.request({
				method: "POST",
				url: "/main.json?cmd=add calendar&uid=" + cookie.getItem("id"),
			}).then(function(res) {
				ctrl.loadConversation()
			}, function(err) {
				console.error(err)
			})
		}
	}
	ctrl.confirmAddCreditCard = function() {
		if (confirm("Are you sure you want to have the user add a credit card?")) {
			m.request({
				method: "POST",
				url: "/main.json?cmd=add credit card&uid=" + cookie.getItem("id"),
			}).then(function(res) {
				ctrl.loadConversation()
			}, function(err) {
				console.error(err)
			})
		}
	}
	ctrl.confirmAddShippingAddr = function() {
		if (confirm("Are you sure you want to have the user add a shipping address?")) {
			m.request({
				method: "POST",
				url: "/main.json?cmd=add shipping address&uid=" + cookie.getItem("id"),
			}).then(function(res) {
				m.route("/train/" + m.route.param("id") +
						"?uid=" + m.route.param("uid"))
			}, function(err) {
				console.error(err)
			})
		}
	}
	ctrl.props = {
		chatWindowsOpen: m.prop(1),
		Username: m.prop(""),
		Addresses: m.prop([]),
		Calendars: m.prop([]),
		Cards: m.prop([]),
		Messages: m.prop([]),
		Preferences: m.prop([]),
		UsernameDisabled: "disabled",
	}
	ctrl.loadConversation()
}
ava.TrainShow.view = function(ctrl) {
	return m(".body", [
		m.component(ava.Header),
		ava.TrainShow.viewFull(ctrl),
		m.component(ava.Footer)
	])
}
ava.TrainShow.viewFull = function(ctrl) {
	return m("#full.container", [
		m(".row", [
			m(".col-md-12", [
				m(".pull-right margin-top-sm", [
					m("a[href=#/]", {
						onclick: ctrl.newChatWindow,
						class: "btn btn-xs"
					}, "New conversation"),
					m("button[href=#/]", {
						onclick: ctrl.confirmComplete,
						class: "btn btn-primary btn-sm margin-left"
					}, "Mark Complete")
				]),
				m("h1", "Training")
			])
		]),
		m(".row.margin-top-sm", [
			m(".col-md-4", m.component(ava.Chatbox, ctrl.props, ctrl)),
			function() {
				var windows = []
				for (var i = 1; i < ctrl.props.chatWindowsOpen(); ++i) {
					windows.push(
						m(".col-md-4", m.component(ava.Chatbox, null, ctrl)))
				}
				return windows
			}(),
			m(".col-md-4.options", [
				m("h4", "Quick Functions"),
				m("ul.list-unstyled", [
					m("li", [
						m("a[href=#/]", {
							onclick: ctrl.toggleCalSelector
						}, "Calendar availability"),
						m("#calendar-selector.margin-sm", {
							class: "hidden"
						}, [
							function() {
								if (ctrl.props.Calendars().length > 0) {
									return m.component(ava.CalendarSelector, ctrl.props)
								}
								return m("div", "Please connect a calendar")
							}()
						])
					]),
					m("li", [
						m("a[href=#/]", {
							onclick: ctrl.toggleUserPrefs
						}, "User preferences"),
						m("#user-prefs.margin-sm", {
							class: "hidden"
						}, [
							m(".margin-left", [
								m("ul.list-unstyled", [
									ctrl.props.Preferences().map(function(item) {
										return m("li", item)
									})
								]),
								m("a.btn.btn-xs[href=#/]", {
									onclick: ctrl.addPreference
								}, "+ New preference")
							])
						])
					]),
					m("li", m("a[href=#/]", "Past purchases")),
					m("li", [
						m("a[href=#/]", {
							onclick: ctrl.toggleContactSearch
						}, "Find contact"),
						m("#contact-search.margin-sm", {
							class: "hidden"
						}, [
							m("input.form-control.form-white", {
								placeholder: "Search contacts"
							})
						])
					]),
					m("li", [
						m("a[href=#/]", {
							onclick: ctrl.toggleAccountInfo
						}, "Edit account info (address, credit card, etc)"),
						m("#edit-account-info.margin-sm", {
							class: "hidden"
						}, [
							m("ul", { class: "list-unstyled" }, [
								m("li", [
									"Credit card: ",
									function() {
										if (ctrl.props.Cards().length > 0) {
											return d.Addresses().map(function(item) {
												return m("span", { class: "green-text" }, item + " ")
											})
										} else {
											return m("span", { class: "red-text" }, "None")
										}
									}(),
									m("a[href=#/]", {
										onclick: ctrl.confirmAddCreditCard.bind(ctrl),
									}, " (Add)"),
								]),
								m("li", [
									"Shipping addresses: ",
									function() {
										if (ctrl.props.Addresses().length > 0) {
											return d.Addresses().map(function(item) {
												return m("span", { class: "green-text" }, item + " ")
											})
										} else {
											return m("span", { class: "red-text" }, "None")
										}
									}(),
									m("a[href=#/]", {
										onclick: ctrl.confirmAddShippingAddr,
									}, " (Add)"),
								]),
								m("li", [
									"Calendar: ",
									function() {
										if (ctrl.props.Calendars().length > 0) {
											return d.Calendars().map(function(item) {
												return m("span", { class: "green-text" }, item + " ")
											})
										} else {
											return m("span", { class: "red-text" }, "None")
										}
									}(),
									m("a[href=#/]", {
										onclick: ctrl.confirmAddCalendar,
									}, " (Add)"),
								])
							])
						])
					])
				]),
				m(".form-group", { class: "hidden" }, [
					m("input.form-control[type=text]", {
						placeholder: "Your message"
					})
				]),
				m("div", { class: "hidden" }, [
					m("input.form-control[type=text]", {
						placeholder: "New preference key",
					}),
					m("input.form-control[type=text]", {
						placeholder: "New preference value",
					}),
					m("input.btn.btn-sm[type=submit]", {
						class: "disabled"
					})
				]),
			])
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
