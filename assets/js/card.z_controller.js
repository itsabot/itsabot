Card.controller = function() {
	var tmp = cookie.getItem("id");
	if (tmp === null || tmp <= 0) {
		return m.route("/login?r=" + encodeURIComponent(window.location.search));
	}
	var _this = this;
	_this.vm = new Card.vm(_this);
	_this.card = new Card({});
	_this.error = m.prop("");
	_this.saveCard = function(ev) {
		ev.preventDefault();
		if (_this.vm.saving()) {
			return;
		}
		_this.vm.save();
		_this.error(_this.vm.validateFields());
		if (_this.error() !== "") {
			_this.vm.saveComplete();
			return;
		}
		_this.card.save().then(function(data) {
			m.route("/profile");
			m.redraw();
		}, function(err) {
			_this.error(err.message);
			_this.vm.saveComplete();
		});
	};
};
