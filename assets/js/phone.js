var Phone = function() {
	var _this = this;
	_this.number = m.prop("");
	_this.format = function() {
		var a1 = _this.number().slice(0, 2);
		var a2 = " (" + _this.number().slice(2, 5) + ") ";
		var a3 = _this.number().slice(5, 8) + "-"
		return a1 + a2 + a3 + _this.number().slice(8)
	};
	return _this;
};
