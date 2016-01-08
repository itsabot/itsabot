var TrainIndex = {
	loadConversations: function(uid) {
		return m.request({
			method: "GET",
			url: "/api/conversations.json"
		});
	},
	route: function(ev) {
		ev.preventDefault();
		m.route("/train/" + ev.currentTarget.attr("key"));
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
		data: TrainIndex.loadConversations(userId)
	};
};

TrainIndex.view = function(ctrl) {
	console.log(ctrl);
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
					ctrl.data.map(function(converation) {
						return m.component(TrainIndexItem, conversation);
					})
				])
			])
		])
	]);
};

var TrainIndexItem = {
	controller: function(args) {
		args.CreatedAt = Date.parse(args.CreatedAt);
		return { conversation: args };
	},
	view: function(ctrl) {
		return m("tr", {
			key: ctrl.conversation.ID,
			onclick: TrainIndex.route
		}, [
				m("td", ctrl.conversation.ID),
				m("td", ctrl.conversation.CreatedAt)
		]);
	}
};

// --- Begin TrainShow

var TrainShow = {
	loadConversation: function(id, uid) {
		return m.request({
			method: "GET",
			url: "/api/conversations.json?uid=" + uid + "&id=" + id
		});
	},
	sendMessage: function(uid) {
		return m.request({
			method: "POST",
			url: "/api/conversation.json?uid=" + uid,
			data: {
				UserID: parseInt(uid, 10),
				Message: TrainIndex.vm.message()
			}
		});
	}
};

TrainShow.controller = function() {
	var id = m.route.param("id");
	var userId = cookie.getItem("id");
	// {
	//		ID, []Chats (sorted), []Packages, []UserPreferences
	// }
	return { data: TrainShow.loadConversation(id, userId) }
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
				class: "col-md-7 margin-top-sm"
			}, [
				m("h3", "Conversation"),
				m("div", [
					m.component(Chatbox, ctrl.data)
				])
			])
			/*
			m("div", {
				class: "col-md-5 margin-top-sm"
			}, [
				m("h3", "Suggested responses"),
				m("h3", "User preferences"),
			])
			*/
		])
	]);
};

var Chatbox = {
	controller: function(args) {
		//		ID, []Chats (sorted), []Packages, []UserPreferences
		return { data: args };
	},
	view: function(ctrl) {
		return m("div", { class: "chat-container" }, [
			m("ol", { class: "chat-box" }, [
				m("li", { class: "chat-user" }, [
					m("div", { class: "messages" }, [
						m("p", "Hi how are you?"),
						m("time datetime='2009-11-13T20:00'", "Timothy • 51 mins")
					])
				]),
				m("li", { class: "chat-ava" }, [
					m("div", { class: "messages" }, [
						m("p", "Bro. I'm chillin"),
						m("time datetime='2009-11-13T20:00'", "37 mins")
					])
				]),
				m("li", { class: "chat-user" }, [
					m("div", { class: "messages" }, [
						m("p", "Yeah man!"),
						m("time datetime='2009-11-13T20:00'", "Timothy • 32 mins")
					])
				]),
				m("li", { class: "chat-user" }, [
					m("div", { class: "messages" }, [
						m("p", "Cool...")
					])
				])
			])
		]);
	}
};

// TODO Suggestion
