var List = function() {
	List.userId = m.prop("");
	List.type = m.prop("");
	List.label = m.prop("");
	List.placeholder = m.prop("");
	List.showAdd = m.prop(true);
	List.data = function() {
		return m.request({
			method: "GET",
			url: "/api/" + List.type() + ".json?uid=" + List.userId()
		});
	};
	List.showForm = function() {
		document.getElementById("add").classList.remove("hidden");
		document.getElementById("add-btn").classList.add("hidden");
	};
	List.hideForm = function() {
		document.getElementById("add").classList.add("hidden");
		document.getElementById("add-btn").classList.remove("hidden");
		document.getElementById("add-input").value = "";
	};
	List.view = function(items) {
		var _this = this;
		if (items === null) {
			items = [];
		}
		return m("div", [
			m("table", {
				class: "table"
			}, [
				m("tbody",
					function() {
						var ret = [];
						for (var i = 0; i < items.length; ++i) {
							var item = items[i];
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
					}()
				)
			]),
			function() {
				if (_this.showAdd()) {
					return m("div", {
						class: "text-right"
					}, [
						m("a", {
							id: "add-btn",
							class: "btn btn-sm",
							onclick: List.showForm
						}, "+Add"),
						m("div", {
							id: "add",
							class: "hidden"
						}, [
							m("input", {
								id: "add-input",
								class: "form-control",
								type: "text",
								placeholder: _this.placeholder()
							}),
							m("div", {
								class: "margin-top-xs"
							}, [
								m("a", {
									class: "btn btn-sm",
									onclick: List.hideForm
								}, "Cancel"),
								m("a", {
									class: "btn btn-primary btn-sm btn-collection"
								}, "Save")
							])
						])
					])
				}
			}()
		]);
	};
	return List;
};
