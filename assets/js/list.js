var List = function(data) {
	var _this = this;
	_this.id = Math.floor((Math.random() * 1000000000) + 1);
	_this.userId = m.prop(cookie.getItem("id"));
	_this.type = m.prop(data.type);
	_this.placeholder = m.prop(data.placeholder || "");
	_this.data = m.prop([]);
	_this.view = function() {
		return m("div", {class: "table-responsive"}, [
			function() {
				return _this.type().listView(_this);
			}()
		]);
	};
};
