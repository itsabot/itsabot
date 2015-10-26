var Trainer = function() {
	this.id = m.prop(0);
	this.sentence = function() {
		return m.request({method: "GET", url: "/api/sentence.json"})
	};
	this.save = function() {
		var sentence = "";
		for (var i = 0; i < train.vm.words.length; ++i) {
			var word = train.vm.words[i];
			if (word.type() === "") {
				word.type("_N(" + word.value() + ")");
			}
			sentence += word.type() + " ";
		}
		var data = {
			Id: train.vm.state.id,
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
			data: {id: id}
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
		switch (train.vm.trainingCategory()) {
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
				train.vm.trainingCategory()
			);
		};
	};
};

var train = {};
train.controller = {
	init: function() {
		train.controller.trainer = new Trainer();
		train.controller.trainer.sentence().then(function(data) {
			if (data.Id === 0) {
				m.render(
					document.getElementById("trainer"),
					train.viewEmpty()
				);
			} else {
				train.vm.init(data);
				m.render(
					document.getElementById("trainer"),
					train.view(train.controller)
				);
			}
		});
	}
};

train.vm = {
	init: function(data) {
		train.vm.trainer = new Trainer();
		var words = data.Sentence.split(/\s+/);
		train.vm.words = [];
		for (var i = 0; i < words.length; ++i) {
			train.vm.words[i] = new Word(words[i]);
		}
		train.vm.trainingCategory = m.prop("COMMANDS");
		train.vm.state = {};
		train.vm.state.id = data.Id;
		train.vm.nextCategory = function() {
			var el = document.getElementById("training-category");
			var helpTitle = document.getElementById("help-title");
			var helpBody = document.getElementById("help-body");
			switch (train.vm.trainingCategory()) {
			case "COMMANDS":
				train.vm.trainingCategory("OBJECTS");
				var btn = document.getElementById("back-btn");
				btn.classList.remove("hidden");
				helpTitle.innerText = "What is an object? ";
				helpBody.innerText = "Objects are the direct objects of the sentence.";
				break;
			case "OBJECTS":
				train.vm.trainingCategory("ACTORS");
				helpTitle.innerText = "What is an actor? ";
				helpBody.innerText = "Actors are often the indirect objects of the sentence.";
				break;
			case "ACTORS":
				train.vm.trainingCategory("TIMES");
				helpTitle.innerText = "What are times? ";
				helpBody.innerText = "Every Tuesday. Noon. Friday. Tomorrow. Etc.";
				break;
			case "TIMES":
				train.vm.trainingCategory("PLACES");
				var btn = document.getElementById("continue-btn");
				helpTitle.innerText = "What are places? ";
				helpBody.innerText = "A place is any description of where an event should take place. Starbucks. Nearby. Etc.";
				btn.innerText = "Save";
				break;
			case "PLACES":
				var btn = document.getElementById("continue-btn");
				if (btn.innerText !== "Saving...") {
					train.controller.trainer.save()
						.then(function() {
							window.location.reload();
						}, console.error("we got probs"));
					btn.innerText = "Saving...";
					btn = document.getElementById("back-btn");
					btn.classList.add("hidden");
				}
				return;
			};
			el.innerText = train.vm.trainingCategory();
			el.className = train.vm.categoryColor();
		};
		train.vm.prevCategory = function() {
			var el = document.getElementById("training-category");
			switch (train.vm.trainingCategory()) {
			case "OBJECTS":
				train.vm.trainingCategory("COMMANDS");
				var btn = document.getElementById("back-btn");
				btn.classList.add("hidden");
				break;
			case "ACTORS":
				train.vm.trainingCategory("OBJECTS");
				break;
			case "TIMES":
				train.vm.trainingCategory("ACTORS");
				break;
			case "PLACES":
				train.vm.trainingCategory("TIMES");
				var btn = document.getElementById("continue-btn");
				btn.innerText = "Continue";
				break;
			};
			el.innerText = train.vm.trainingCategory();
			el.className = train.vm.categoryColor();
		};
		train.vm.categoryColor = function() {
			switch (train.vm.trainingCategory()) {
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

train.view = function(controller) {
	return m("div", {class:"row"}, [
		m("div", {class: "col-sm-12 text-right"}, [
			m("a", {
				class: "btn",
				onclick: train.controller.trainer.skip
			}, "Skip Sentence"),
		]),
		m("div", {class: "col-sm-12 margin-top-sm"}, [
			m("h2",
				"Tap the ",
				m("span", {
					id: "training-category",
					class: train.vm.categoryColor()
				}, train.vm.trainingCategory()),
				" in this sentence:"
			),
			m("p", {class: "light"}, [
				m("strong",
					m("i", 
						{id: "help-title"},
						"What is a command? "
					)
				),
				m("span",
					{id: "help-body"},
					'A command is a verb, like "Find," "Walk," or "Meet."'
				)
			])
		]),
		m("div", {class: "col-sm-12"}, [
			m("p", { id: "train-sentence", class: "big no-select" }, [
				train.vm.words.map(function(word, i) {
					return [
						m("span", {
							onclick: word.setClass,
						}, word.value()),
						m("span", " ")
					]
				})
			])
		]),
		m("div", {class: "col-sm-12 text-right"}, [
			m("a", {
				id: "back-btn",
				href: "#/",
				class: "btn hidden",
				onclick: train.vm.prevCategory
			}, "Go back"),
			m("a", {
				id: "continue-btn",
				href: "#/",
				class: "btn btn-primary btn-lg",
				onclick: train.vm.nextCategory
			}, "Continue")
		])
	]);
};

train.viewEmpty = function() {
	return m("div", {class: "col-sm-12 text-center"}, [
		m("h2", "All done!"),
		m("p", "No tasks need to be completed.")
	]);
};

window.onload = function() {
	train.controller.init();
	window.addEventListener("keypress", function(ev) {
		if (ev.keyCode === 102 /* 'f' key */) {
			ev.preventDefault();
			train.vm.nextCategory();
		}
	});
};
