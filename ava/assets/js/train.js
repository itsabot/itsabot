var Trainer = function() {
	this.id = m.prop(0);
	this.sentence = function() {
		return m.request({
			method: "GET",
			url: "/api/sentence.json"
		})
	};
	this.save = function() {
		var sentence = "";
		for (var i = 0; i < Train.vm.words.length; ++i) {
			var word = Train.vm.words[i];
			if (word.type() === "") {
				word.type("_N(" + word.value() + ")");
			}
			sentence += word.type() + " ";
		}
		var data = {
			Id: Train.vm.state.id,
			Sentence: sentence
		};
		return m.request({
			method: "PUT",
			url: "/api/sentence.json",
			data: data
		});
	};
	this.skip = function(id) {
		return m.request({
			method: "GET",
			url: "/api/sentence.json",
			data: {
				id: id
			}
		});
	};
};

var Word = function(word) {
	var _this = this;
	this.value = m.prop(word);
	this.type = m.prop("");
	this.setClass = function() {
		if (this.classList.length > 0) {
			return this.className = "";
		}
		switch (Train.vm.trainingCategory()) {
			case "COMMANDS":
				_this.type("_C(" + _this.value() + ")");
				return this.classList.add("red");
			case "OBJECTS":
				_this.type("_O(" + _this.value() + ")");
				return this.classList.add("blue");
			case "ACTORS":
				_this.type("_A(" + _this.value() + ")");
				return this.classList.add("green");
			case "TIMES":
				_this.type("_T(" + _this.value() + ")");
				return this.classList.add("yellow");
			case "PLACES":
				_this.type("_P(" + _this.value() + ")");
				return this.classList.add("pink");
			default:
				console.error(
					"invalid training state: " +
					Train.vm.trainingCategory()
				);
		};
	};
};

var Train = {};
Train.controller = {
	init: function() {
		Train.controller.trainer = new Trainer();
		Train.controller.trainer.sentence().then(function(data) {
			if (data.Id === 0) {
				m.render(
					document.getElementById("trainer"),
					Train.viewEmpty()
				);
			} else {
				Train.vm.init(data);
				m.render(
					document.getElementById("trainer"),
					Train.view(Train.controller)
				);
			}
		});
	}
};

Train.vm = {
	init: function(data) {
		Train.vm.trainer = new Trainer();
		var words = data.Sentence.split(/\s+/);
		Train.vm.words = [];
		for (var i = 0; i < words.length; ++i) {
			Train.vm.words[i] = new Word(words[i]);
		}
		Train.vm.trainingCategory = m.prop("COMMANDS");
		Train.vm.state = {};
		Train.vm.state.id = data.Id;
		Train.vm.nextCategory = function() {
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
					helpBody.innerText = "Every Tuesday. Noon. Friday. Tomorrow. Etc.";
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
					if (btn.innerText !== "Saving...") {
						Train.controller.trainer.save()
							.then(function() {
								window.location.reload();
							}, console.error("we got probs"));
						btn.innerText = "Saving...";
						btn = document.getElementById("back-btn");
						btn.classList.add("hidden");
					}
					return;
			};
			el.innerText = Train.vm.trainingCategory();
			el.className = Train.vm.categoryColor();
		};
		Train.vm.prevCategory = function() {
			var el = document.getElementById("training-category");
			switch (Train.vm.trainingCategory()) {
				case "OBJECTS":
					Train.vm.trainingCategory("COMMANDS");
					var btn = document.getElementById("back-btn");
					btn.classList.add("hidden");
					break;
				case "ACTORS":
					Train.vm.trainingCategory("OBJECTS");
					break;
				case "TIMES":
					Train.vm.trainingCategory("ACTORS");
					break;
				case "PLACES":
					Train.vm.trainingCategory("TIMES");
					var btn = document.getElementById("continue-btn");
					btn.innerText = "Continue";
					break;
			};
			el.innerText = Train.vm.trainingCategory();
			el.className = Train.vm.categoryColor();
		};
		Train.vm.categoryColor = function() {
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
		};
	}
};

Train.view = function(controller) {
	return m("div", {
		class: "row"
	}, [
		m("div", {
			class: "col-sm-12 text-right"
		}, [
			m("a", {
				class: "btn",
				onclick: Train.controller.trainer.skip
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
	]);
};

Train.viewEmpty = function() {
	return m("div", {
		class: "col-sm-12 text-center"
	}, [
		m("h2", "All done!"),
		m("p", "No tasks need to be completed.")
	]);
};

window.onload = function() {
	Train.controller.init();
	window.addEventListener("keypress", function(ev) {
		if (ev.keyCode === 102 /* 'f' key */ ) {
			ev.preventDefault();
			Train.vm.nextCategory();
		}
	});
};
