var Phone = function() {
	Phone.number = m.prop("");
	Phone.format = function() {
		var a1 = Phone.number().slice(0, 2);
		var a2 = " (" + Phone.number().slice(2, 5) + ") ";
		var a3 = Phone.number().slice(5, 8) + "-"
		return a1 + a2 + a3 + Phone.number().slice(8)
	};
	return Phone;
};
