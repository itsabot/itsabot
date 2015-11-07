var List = function() {
	var _this = this;
	this.id = Math.floor((Math.random() * 1000000000) + 1);
	this.userId = m.prop(cookie.getItem("id"));
	this.type = m.prop("");
	this.placeholder = m.prop("");
	this.showAdd = m.prop(true);
	this.data = m.prop([]);
	this.showForm = function() {
		document.getElementById(_this.id + "-add").classList.remove("hidden");
		document.getElementById(_this.id + "-add-btn").classList.add("hidden");
	};
	this.hideForm = function() {
		document.getElementById(_this.id + "-add").classList.add("hidden");
		document.getElementById(_this.id + "-add-btn").classList.remove("hidden");
		document.getElementById(_this.id + "-add-input").value = "";
	};
	this.view = function() {
		// non-strict comparison detects null and undefined
		if (_this.data() == null) {
			_this.data([]);
		}
		return m("div", [
			m("table", {
				class: "table"
			}, [
				m("tbody", [

					function() {
						if (_this.type() === "phones") {
							return List.phoneList.listView(_this);
						}
						if (_this.type() === "cards") {
							return List.cardList.listView(_this);
						}
					}()
				])
			]),
			function() {
				if (_this.showAdd()) {
					return m("div", {
						class: ""
					}, [

						function() {
							if (_this.type() === "phones") {
								return List.phoneList.addView(_this);
							}
							if (_this.type() === "cards") {
								return List.cardList.addView(_this);
							}
						}()
					])
				}
			}()
		]);
	};
};

List.phoneList = {
	listView: function(list) {
		var ret = [];
		for (var i = 0; i < list.data().length; ++i) {
			var item = list.data()[i];
			var phone = new Phone();
			phone.number(item.Number);
			var v = m("tr", {
				"attr-id": item.Id
			}, [
				m("td", phone.format()),
				m("td", {
					class: "text-right"
				}, [
					m("img", {
						class: "icon icon-xs icon-delete",
						src: "/public/images/icon_delete.svg",
						onclick: function() {
							var c = confirm("Delete this number?");
							if (c) {
								// TODO delete from database
								console.log("not implemented");
							}
						}
					})
				])
			]);
			ret.push(v);
		}
		return ret;
	},
	addView: function(list) {
		return m("div", [
			m("a", {
				id: list.id + "-add-btn",
				class: "btn btn-sm",
				onclick: list.showform
			}, "+add"),
			m("div", {
				id: list.id + "-add",
				class: "hidden"
			}, [
				m("input", {
					id: list.id + "-add-input",
					class: "form-control",
					type: "text",
					placeholder: list.placeholder()
				}),
				m("div", {
					class: "margin-top-xs"
				}, [
					m("a", {
						class: "btn btn-sm",
						onclick: list.hideForm
					}, "Cancel"),
					m("a", {
						class: "btn btn-primary btn-sm btn-collection"
					}, "Save")
				])
			])
		]);
	}
};

List.cardList = {
	listView: function(list) {
		var ret = [];
		for (var i = 0; i < list.data().length; ++i) {
			var item = list.data()[i];
			var card = new Card();
			card.id(item.Id);
			card.last4(item.Last4);
			card.expMonth(item.ExpMonth);
			card.expYear(item.ExpYear);
			card.brand(item.Brand);
			var v = m("tr", {
				"attr-id": item.Id
			}, [
				m("td", card.last4()),
				m("td", card.expMonth()),
				m("td", card.expYear()),
				m("td", card.brand()),
				m("td", {
					class: "text-right"
				}, [
					m("img", {
						class: "icon icon-xs icon-delete",
						src: "/public/images/icon_delete.svg",
						onclick: function() {
							var c = confirm("Delete this number?");
							if (c) {
								// TODO delete from database
								console.log("not implemented");
							}
						}
					})
				])
			]);
			ret.push(v);
		}
		return ret;
	},
	addView: function(list) {
		return m("div", [
			m("a", {
				id: list.id + "-add-btn",
				class: "btn btn-sm",
				onclick: list.showForm
			}, "+Add"),
			m("div", {
				id: list.id + "-add",
				class: "hidden text-left"
			}, [
				m("div", {
					class: "form-horizontal"
				}, [
					m("div", {
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-4"
						}, "Card number"),
						m("div", {
							class: "col-md-8"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "5555 5555 5555 1234"
							})
						])
					]),
					m("div", {
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-4"
						}, "Expires"),
						m("div", {
							class: "col-md-8"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "01 / 2015"
							})
						]),
					]),
					m("div", {
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-4"
						}, "CVC"),
						m("div", {
							class: "col-md-8"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "123"
							})
						])
					]),
					m("div", {
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-4"
						}, "Cardholder name"),
						m("div", {
							class: "col-md-8"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "Cardholder name"
							})
						])
					])
				]),
				m("div", {
					class: "margin-top-xs text-right"
				}, [
					m("a", {
						class: "btn btn-sm",
						onclick: list.hideForm
					}, "Cancel"),
					m("a", {
						class: "btn btn-primary btn-sm btn-collection"
					}, "Save")
				])
			])
		]);
	}
};
