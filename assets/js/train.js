var Trainer = function() {
	var _this = this;
	_this.sentence = function(id) {
		var url = "/api/sentence.json";
		if (id !== undefined) {
			url += "?id=" + id;
		}
		return m.request({
			method: "GET",
			url: url
		})
	};
	_this.save = function() {
		var sentence = "";
		for (var i = 0; i < Train.vm.words.length; ++i) {
			var word = Train.vm.words[i];
			sentence += word.type() + " ";
		}
		var data = {
			ID: Train.vm.state.id,
			AssignmentID: Train.vm.state.assignmentId,
			ForeignID: Train.vm.state.foreignId,
			Sentence: sentence,
			MaxAssignments: Train.vm.state.maxAssignments
		};
		return m.request({
			method: "PUT",
			url: "/api/sentence.json",
			data: data
		});
	};
	return this;
};

var Word = function(word) {
	var _this = this;
	_this.value = m.prop(word);
	_this.type = m.prop("_N(" + _this.value() + ")");
	_this.setClass = function() {
		var el = this;
		if (el.classList.length > 0) {
			_this.type("_N(" + _this.value() + ")");
			return el.className = "";
		}
		switch (Train.vm.trainingCategory()) {
			case "COMMANDS":
				_this.type("_C(" + _this.value() + ")");
				return el.classList.add("red");
			case "OBJECTS":
				_this.type("_O(" + _this.value() + ")");
				return el.classList.add("blue");
			case "ACTORS":
				_this.type("_A(" + _this.value() + ")");
				return el.classList.add("green");
			case "TIMES":
				_this.type("_T(" + _this.value() + ")");
				return el.classList.add("yellow");
			case "PLACES":
				_this.type("_P(" + _this.value() + ")");
				return el.classList.add("pink");
			default:
				console.error(
					"invalid training state: " +
					Train.vm.trainingCategory()
				);
		};
	};
};

var Train = {};
Train.controller = function() {
	var id = m.route.param("sentenceID");
	var _this = this;
	_this.trainer = new Trainer();
	_this.trainer.sentence(id).then(function(data) {
		Train.vm.init(_this, data);
	});
	return this;
};

Train.vm = {
	init: function(controller, data) {
		Train.vm.state = {};
		Train.vm.trainer = new Trainer();
		Train.vm.trainingCategory = m.prop("COMMANDS");
		Train.vm.state.id = data.ID;
		Train.vm.state.assignmentId = m.route.param("assignmentId");
		Train.vm.state.foreignId = data.ForeignID;
		Train.vm.state.maxAssignments = data.MaxAssignments;
		Train.vm.words = [];
		var words = data.Sentence.split(/\s+/);
		for (var i = 0; i < words.length; ++i) {
			Train.vm.words[i] = new Word(words[i]);
		}
		window.addEventListener("keypress", function(ev) {
			if (ev.keyCode === 102 /* 'f' key */ ) {
				ev.preventDefault();
				Train.vm.nextCategory(controller);
			} else if (ev.keyCode === 98 /* 'b' key */ ) {
				ev.preventDefault();
				Train.vm.prevCategory();
			}
		});
	},
	nextCategory: function(controller) {
		var el = document.getElementById("training-category");
		var helpTitle = document.getElementById("help-title");
		var helpBody = document.getElementById("help-body");
		switch (Train.vm.trainingCategory()) {
			case "COMMANDS":
				Train.vm.trainingCategory("OBJECTS");
				var btn = document.getElementById("back-btn");
				btn.classList.remove("hidden");
				helpTitle.innerText = "What is an object? ";
				helpBody.innerText = "Objects are the direct objects of the sentence.";
				break;
			case "OBJECTS":
				Train.vm.trainingCategory("ACTORS");
				helpTitle.innerText = "What is an actor? ";
				helpBody.innerText = "Actors are often the indirect objects of the sentence.";
				break;
			case "ACTORS":
				Train.vm.trainingCategory("TIMES");
				helpTitle.innerText = "What are times? ";
				helpBody.innerText = "Every Tuesday. Noon. Friday. Tomorrow. This Wednesday. Etc.";
				break;
			case "TIMES":
				Train.vm.trainingCategory("PLACES");
				var btn = document.getElementById("continue-btn");
				helpTitle.innerText = "What are places? ";
				helpBody.innerText = "A place is any description of where an event should take place. Starbucks. Nearby. Etc.";
				btn.innerText = "Save";
				break;
			case "PLACES":
				var btn = document.getElementById("continue-btn");
				if (btn.innerText !== "Saving..." && btn.innerText !== "Thank you!") {
					Train.vm.save();
					console.log(controller);
					controller.trainer.save().then(function() {
						Train.vm.saveComplete();
						setTimeout(function() {
							m.route("/train");
						}, 2000);
					});
				}
				return;
		};
		el.innerText = Train.vm.trainingCategory();
		el.className = Train.vm.categoryColor();
	},
	prevCategory: function() {
		var el = document.getElementById("training-category");
		var helpTitle = document.getElementById("help-title");
		var helpBody = document.getElementById("help-body");
		switch (Train.vm.trainingCategory()) {
			case "OBJECTS":
				Train.vm.trainingCategory("COMMANDS");
				helpTitle.innerText = "What is a command? ";
				helpBody.innerText = 'A command is a verb, like "Find," "Walk," or "Meet."';
				var btn = document.getElementById("back-btn");
				btn.classList.add("hidden");
				break;
			case "ACTORS":
				Train.vm.trainingCategory("OBJECTS");
				helpTitle.innerText = "What is an object? ";
				helpBody.innerText = "Objects are the direct objects of the sentence.";
				break;
			case "TIMES":
				Train.vm.trainingCategory("ACTORS");
				helpTitle.innerText = "What is an actor? ";
				helpBody.innerText = "Actors are often the indirect objects of the sentence.";
				break;
			case "PLACES":
				Train.vm.trainingCategory("TIMES");
				helpTitle.innerText = "What are times? ";
				helpBody.innerText = "Every Tuesday. Noon. Friday. Tomorrow. This Wednesday. Etc.";
				var btn = document.getElementById("continue-btn");
				btn.innerText = "Continue";
				break;
		};
		el.innerText = Train.vm.trainingCategory();
		el.className = Train.vm.categoryColor();
	},
	categoryColor: function() {
		switch (Train.vm.trainingCategory()) {
			case "COMMANDS":
				return "red";
			case "OBJECTS":
				return "blue";
			case "ACTORS":
				return "green";
			case "TIMES":
				return "yellow";
			case "PLACES":
				return "pink";
		};
	},
	save: function() {
		var btn = document.getElementById("continue-btn");
		btn.innerText = "Saving...";
		btn = document.getElementById("back-btn");
		btn.classList.add("hidden");
	},
	saveComplete: function() {
		var btn = document.getElementById("continue-btn");
		btn.innerText = "Thank you!";
	}
};

Train.view = function(controller) {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Train.viewFull(),
		Train.viewEmpty(),
		Footer.view()
	]);
};

Train.viewFull = function() {
	var view = m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-sm-12"
			}, [
				m("h1", "Train"),
				m("p", "Train Ava to understand language.")
			])
		]),
		m("div", {
			class: "row"
		}, [
			m("div", {
				class: "col-sm-12 text-right"
			}, [
				m("a", {
					class: "btn",
					onclick: function() {
						m.route("/train");
					}
				}, "Skip Sentence"),
			]),
			m("div", {
				class: "col-sm-12 margin-top-sm"
			}, [
				m("h2",
					"Tap the ",
					m("span", {
						id: "training-category",
						class: Train.vm.categoryColor()
					}, Train.vm.trainingCategory()),
					" in this sentence:"
				),
				m("p", {
					class: "light"
				}, [
					m("strong",
						m("i", {
							id: "help-title"
						}, "What is a command? ")
					),
					m("span", {
						id: "help-body"
					}, 'A command is a verb, like "Find," "Walk," or "Meet."')
				])
			]),
			m("div", {
				class: "col-sm-12"
			}, [
				m("p", {
					id: "train-sentence",
					class: "big no-select"
				}, [
					Train.vm.words.map(function(word, i) {
						return [
							m("span", {
								onclick: word.setClass,
							}, word.value()),
							m("span", " ")
						]
					})
				])
			]),
			m("div", {
				class: "col-sm-12 text-right"
			}, [
				m("a", {
					id: "back-btn",
					href: "#/",
					class: "btn hidden",
					onclick: Train.vm.prevCategory
				}, "Go back"),
				m("a", {
					id: "continue-btn",
					href: "#/",
					class: "btn btn-primary btn-lg",
					onclick: Train.vm.nextCategory
				}, "Continue")
			])
		])
	]);
	if (Train.vm.state.id > 0) {
		return view;
	}
};

Train.viewEmpty = function() {
	var view = m("div", {
		id: "empty",
		class: "container jumbo"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-sm-12 text-center"
			}, [
				m("h2", "All done!"),
				m("p", {
					style: "color:black"
				}, "No tasks need to be completed.")
			])
		])
	]);
	if (Train.vm.state.id === 0 || Train.vm.state.id === undefined) {
		return view;
	}
};
