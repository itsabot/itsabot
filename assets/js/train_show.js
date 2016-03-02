(function(abot) {
abot.TrainShow = {}
abot.TrainShow.controller = function() {
	if (!abot.isLoggedIn()) {
		m.route("/login?r=" + encodeURIComponent(window.location.search))
		return
	}
	if (!abot.isTrainer()) {
		m.route("/profile")
		return
	}
	var uri = 'ws:'
	if (window.location.protocol === 'https:') {
		uri = 'wss:'
	}
	var uid = parseInt(cookie.getItem("id"))
	uri += '//' + window.location.host + '/ws?UserID=' + uid
	var sockInterval
	var ctrl = this
	ctrl.connectSocket = function() {
		abot.socket = new WebSocket(uri)
		abot.socket.onopen = function() {
			console.log("opened socket")
			clearInterval(sockInterval)
		}
		abot.socket.onmessage = function(ev) {
			console.log("message received")
			try { var msgs = JSON.parse(ev.data) } catch(err) { return }
			for (var i = 0; i < msgs.length; ++i) {
				ctrl.props.Messages().push(msgs[i])
			}
			m.redraw(true)
		}
		abot.socket.onclose = function() {
			console.log("socket closed, setting retry interval")
			sockInterval = setInterval(function() {
				console.log("retrying socket...")
				ctrl.connectSocket()
			}, 5000)
		}
	}
	ctrl.connectSocket()
	id = m.route.param("id")
	uid = cookie.getItem("id")
	ctrl.loadConversation = function() {
		abot.request({
			method: "GET",
			url: "/api/trainer/message.json?id=" + id
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
			abot.request({
				method: "PATCH",
				url: "/api/trainer/conversation.json?uid=" + userId,
			}).then(function() {
				m.route("/train?trained=true")
			}, function(err) {
				console.error(err)
			})
		}
	}
	ctrl.confirmAddCalendar = function() {
		if (confirm("Are you sure you want to have the user add a calendar?")) {
			abot.request({
				method: "POST",
				url: "/api/trainer/trigger.json?cmd=add calendar&uid=" + cookie.getItem("id"),
			}).then(function(res) {
				ctrl.loadConversation()
			}, function(err) {
				console.error(err)
			})
		}
	}
	ctrl.confirmAddCreditCard = function() {
		if (confirm("Are you sure you want to have the user add a credit card?")) {
			abot.request({
				method: "POST",
				url: "/api/trainer/trigger.json?cmd=add credit card&uid=" + cookie.getItem("id"),
			}).then(function(res) {
				ctrl.loadConversation()
			}, function(err) {
				console.error(err)
			})
		}
	}
	ctrl.confirmAddShippingAddr = function() {
		if (confirm("Are you sure you want to have the user add a shipping address?")) {
			abot.request({
				method: "POST",
				url: "/api/trainer/trigger.json?cmd=add shipping address&uid=" + cookie.getItem("id"),
			}).then(function(res) {
				m.route("/train/" + m.route.param("id") +
						"?uid=" + cookie.getItem("id"))
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
		UsernameDisabled: "disabled",
	}
	ctrl.loadConversation()
}
abot.TrainShow.view = function(ctrl) {
	return m(".main", [
		m.component(abot.Header),
		abot.TrainShow.viewFull(ctrl),
	])
}
abot.TrainShow.viewFull = function(ctrl) {
	return m("#full", [
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
		m("h1", "Training"),
		m(".row.margin-top-sm", [
			m.component(abot.Chatbox, ctrl.props, ctrl),
			function() {
				var windows = []
				for (var i = 1; i < ctrl.props.chatWindowsOpen(); ++i) {
					windows.push(
						m(".col-md-4", m.component(abot.Chatbox, null, ctrl)))
				}
				return windows
			}(),
			m(".train-show-options", [
				m("h4.train-show-options-title", "Quick Functions"),
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
									return m.component(abot.CalendarSelector, ctrl.props)
								}
								return m("div", "Please connect a calendar")
							}()
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
				])
			])
		])
	])
}
})(!window.abot ? window.abot={} : window.abot);
