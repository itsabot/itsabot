var TrainIndex = {
	loadConversations: function() {
		return m.request({
			method: "GET",
			url: "/api/conversations.json"
		});
	},
	route: function(ev) {
		ev.preventDefault();
		var id = ev.target.parentNode.getAttribute("data-id");
		var uid = ev.target.parentNode.getAttribute("data-user-id");
		m.route("/train/" + id + "?uid=" + uid);
	}
};
TrainIndex.controller = function() {
	var userId = cookie.getItem("id");
	return {
		// [
		//		{
		//			ID: 200,
		//			Title: "Find me a wine",
		//			CreatedAt: datetime
		//		},
		//		{
		//			...
		//		}
		// ]
		data: TrainIndex.loadConversations()
	};
};
TrainIndex.view = function(ctrl) {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		TrainIndex.viewFull(ctrl),
		Footer.view()
	]);
};
TrainIndex.viewFull = function(ctrl) {
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
				m("h1", "Training")
			])
		]),
		m("div", {
			class: "row"
		}, [
			m("div", {
				class: "col-md-12 margin-top-sm"
			}, [
				m("table", {
					class: "table table-bordered table-hover"
				}, [
					m("tbody", [
						ctrl.data().map(function(conversation) {
							return m.component(TrainIndexItem, conversation);
						})
					])
				])
			])
		])
	]);
};

var TrainIndexItem = {
	controller: function(args) {
		return { conversation: args };
	},
	view: function(ctrl) {
		var t = prettyDate(ctrl.conversation.CreatedAt);
		return m("tr", {
			"data-id": ctrl.conversation.ID,
			"data-user-id": ctrl.conversation.UserID,
			key: ctrl.conversation.ID,
			onclick: TrainIndex.route
		}, [
				m("td", ctrl.conversation.Sentence),
				m("td", t)
		]);
	}
};

var TrainShow = {
	loadConversation: function(id, uid) {
		return m.request({
			method: "GET",
			url: "/api/conversation.json?id=" + id + "&uid=" + uid
		});
	}
};
TrainShow.controller = function() {
	var ctrl = this;
	var id = m.route.param("id");
	var userId = m.route.param("uid");
	ctrl.chatWindowsOpen = m.prop(1);
	ctrl.data = TrainShow.loadConversation(id, userId);
	// {
	//		ID, []Chats (sorted), []Packages, []UserPreferences
	// }
	return ctrl;
};
TrainShow.view = function(ctrl) {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		TrainShow.viewFull(ctrl),
		Footer.view()
	]);
};
TrainShow.viewFull = function(ctrl) {
	return m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row"
		}, [
			m("div", {
				class: "col-md-12"
			}, [
				m("div", {
					class: "pull-right margin-top-sm"
				}, [
					m("a[href=#/]", {
						onclick: TrainShow.newChatWindow.bind(ctrl),
						class: "btn btn-xs"
					}, "New conversation"),
					m("button[href=#/]", {
						onclick: TrainShow.confirmComplete,
						class: "btn btn-primary btn-sm margin-left"
					}, "Mark Complete")
				]),
				m("h1", "Training")
			])
		]),
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-md-4",
			}, [
				m("div", [
					m.component(Chatbox, ctrl.data())
				]),
			]),
			function() {
				var chats = [];
				for (var i = 1; i < ctrl.chatWindowsOpen(); ++i) {
					m("div", [
						chats.push(
							m("div", {
								class: "col-md-4"
							}, m.component(Chatbox))
						)
					]);
				}
				return chats;
			}(),
			m("div", {
				class: "col-md-4 options"
			}, [
				m("h4", "Quick Functions"),
				m("ul", { class: "list-unstyled" }, [
					m("li", [
						m("a[href=#/]", {
							onclick: TrainShow.vm.toggleCalSelector
						}, "Calendar availability"),
						m("div", {
							id: "calendar-selector",
							class: "margin-sm hidden"
						}, [
							m.component(CalendarSelector, ctrl.data())
						])
					]),
					m("li", [
						m("a[href=#/]", {
							onclick: TrainShow.vm.toggleUserPrefs
						}, "User preferences"),
						m("div", {
							id: "user-prefs",
							class: "hidden margin-sm"
						}, [
							m("div", { class: "margin-left" }, [
								m("ul", { class: "list-unstyled" }, [
									ctrl.data().Preferences.map(function(item) {
										return m("li", item);
									})
								]),
								m("a[href=#/]", {
									class: "btn btn-xs",
									onclick: TrainShow.addPreference
								}, "+ New preference")
							])
						])
					]),
					m("li", [
						m("a[href=#/]", "Past purchases")
					]),
					m("li", [
						m("a[href=#/]", {
							onclick: TrainShow.vm.toggleContactSearch
						}, "Find contact"),
						m("div", {
							id: "contact-search",
							class: "hidden margin-sm"
						}, [
							m("input", {
								class: "form-control form-white",
								placeholder: "Search contacts"
							})
						])
					]),
					m("li", [
						m("a[href=#/]", {
							onclick: TrainShow.vm.toggleAccountInfo
						}, "Edit account info (address, credit card, etc)"),
						m("div", {
							id: "edit-account-info",
							class: "hidden margin-sm"
						}, [
							m("ul", { class: "list-unstyled" }, [
								m("li", [
									"Credit card: ",
									m("span", { class: "green-text" }, "Amex (9999)"),
									m("a[href=#/]", {
										onclick: TrainShow.confirmAddCreditCard,
									}, " (Add)"),
								]),
								m("li", [
									"Shipping addresses: ",
									m("span", { class: "green-text" }, "Home (1418 7th St)"),
									m("a[href=#/]", {
										onclick: TrainShow.confirmAddShippingAddr,
									}, " (Add)"),
								]),
								m("li", [
									"Calendar: ",
									m("span", { class: "red-text" }, "None"),
									m("a[href=#/]", {
										onclick: TrainShow.confirmAddCalendar,
									}, " (Add)"),
								])
							])
						])
					])
				]),
				m("div", { class: "form-group hidden" }, [
					m("input", {
						type: "text",
						class: "form-control",
						placeholder: "Your message"
					})
				]),
				m("div", { class: "hidden" }, [
					m("input", {
						type: "text",
						placeholder: "New preference key",
						class: "form-control"
					}),
					m("input", {
						type: "text",
						placeholder: "New preference value",
						class: "form-control"
					}),
					m("input", {
						type: "submit",
						class: "btn btn-sm disabled"
					})
				]),
			])
		])
	]);
};
TrainShow.vm = {
	toggleCalSelector: function() {
		document.getElementById("calendar-selector").classList.toggle("hidden");
	},
	toggleUserPrefs: function() {
		document.getElementById("user-prefs").classList.toggle("hidden");
	},
	toggleContactSearch: function() {
		document.getElementById("contact-search").classList.toggle("hidden");
	},
	toggleAccountInfo: function() {
		document.getElementById("edit-account-info").classList.toggle("hidden");
	}
};
TrainShow.newChatWindow = function() {
	this.chatWindowsOpen(this.chatWindowsOpen()+1);
};

var Chatbox = {};
Chatbox.controller = function(args) {
	//		ID, []Chats (sorted), []Packages, []UserPreferences
	args = args || { Conversations: [] };
	return {
		Username: args.Username,
		Conversations: args.Conversations
	};
};
Chatbox.view = function(ctrl) {
	return m("div", { class: "chat-container" }, [
		m("input", {
			type: "text",
			class: "disabled",
			placeholder: "To:",
			value: ctrl.Username || ""
		}),
		m("ol", {
			id: "chat-box",
			class: "chat-box",
			config: Chatbox.scrollToBottom
		}, [
			ctrl.Conversations.map(function(c) {
				var d = {
					Username: ctrl.Username,
					Sentence: c.Sentence,
					AvaSent: c.AvaSent,
					CreatedAt: c.CreatedAt
				};
				return m.component(ChatboxItem, d);
			})
		]),
		m("textarea", {
			class: "chat-textarea",
			rows: 4,
			placeholder: "Your message",
			onkeydown: Chatbox.handleSend
		})
	]);
};
/*
Chatbox.vm = {
	showAvaMsg: function(sentence) {}
};
*/
Chatbox.handleSend = function(ev) {
	if (ev.keyCode === 13 /* enter */ && !ev.shiftKey) {
		ev.preventDefault();
		var sentence = ev.srcElement.value;
		// TODO provide some security here ensuring the user has an open
		// needsTraining message, preventing the trainer from swapping the
		// userid to send messages to new users. OR create a tmp uuid to ID the
		// user that's cycled every so often
		Chatbox.send(m.route.param("uid"), sentence).then(function() {
			// TODO there's a better way to do this...
			m.route("/train/" + m.route.param("id") +
					"?uid=" + m.route.param("uid"));
			//Chatbox.vm.showAvaMsg(sentence);
		}, function(err) {
			// TODO display error to the user
			console.error(err);
			ev.srcElement.value = sentence;
		});
		ev.srcElement.value = "";
		return;
	}
};
Chatbox.scrollToBottom = function(ev) {
	ev.scrollTop = ev.scrollHeight;
};
Chatbox.send = function(uid, sentence) {
	return m.request({
		method: "POST",
		url: "/api/conversations.json",
		data: {
			UserID: parseInt(uid),
			Sentence: sentence
		}
	});
};

var ChatboxItem = {
	controller: function(args) {
		return {
			Sentence: args.Sentence,
			CreatedAt: args.CreatedAt,
			Username: args.Username,
			AvaSent: args.AvaSent
		};
	},
	view: function(ctrl) {
		var c, u;
		if (ctrl.AvaSent) {
			c = "chat-ava";
			u = "Ava";
		} else {
			c = "chat-user";
			u = ctrl.Username;
		}
		return m("li", { class: c }, [
			m("div", { class: "messages" }, [
				m("p", ctrl.Sentence),
				m("time", {
					datetime: ctrl.CreatedAt
				}, u + " • " + prettyDate(ctrl.CreatedAt))
			])
		]);
	}
};

var CalendarSelector = {
	controller: function(args) {
		return {
			// {
			//     start timestamp
			//     length (mins)
			// }
			BlockedTimes: args.BlockedTimes
		};
	},
	view: function(ctrl) {
		var t = new Date();
		return m("div", [
			// TODO convert to subcomponent (date selector)
			m("a[href=#/]", { class: "pull-left" }, "<"),
			m("a[href=#/]", { class: "pull-right" }, ">"),
			m("div", { class: "centered" }, [
				function() {
					var d = new Date();
					return t.toLocaleDateString("en-US", {
						weekday: "long",
						year: "numeric",
						month: "numeric",
						day: "numeric",
					});
				}()
			]),
			m("table", { class: "calendar-selector table table-bordered" }, [
				function() {
					var trs = [];
					var hrs = [];
					var colors = [];
					for (var i = t.getHours(), j = 0; i <= 24 && j < 12; ++i && ++j) {
						var c = "";
						if (i <= 8 || i > 17) {
							c = "gray";
						}
						var time = "";
						if (i > 12) {
							time = "" + i - 12;	
						} else {
							time = "" + i;
						}
						if (i < 12) {
							time += "a";
						} else {
							time += "p";
						}
						hrs.push(m("td", { class: c + " calendar-row" }, time));
						colors.push(m("td", { class: c + " calendar-time" }));
					}
					trs.push(m("tr", hrs));
					trs.push(m("tr", colors));
					return trs;
				}()
			]),
			m("div", { class: "subtle subtle-sm pull-right" }, "Timezone: Pacific") 
		]);
	}
};

// TODO Suggestion
//
//
