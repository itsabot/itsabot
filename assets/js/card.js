var Card = function(data) {
	var _this = this;
	data = data || {};
	_this.id = m.prop(data.id || 0);
	_this.cardholderName = m.prop(data.cardholderName || "");
	_this.number = m.prop(data.number || "");
	_this.zip5 = m.prop(data.zip5 || "");
	_this.brand = m.prop("");
	if (data.expMonth != null && data.expYear != null) {
		_this.expiry = m.prop(data.expMonth + " / " + data.expYear);
	} else {
		_this.expiry = m.prop(data.expiry || "");
	}
	_this.cvc = m.prop(data.cvc || "");
	_this.last4 = m.prop(data.last4 || "");
	_this.save = function() {
		var deferred = m.deferred();
		saveStripe().then(function(resp) {
			_this.brand(resp.card.brand);
			var data = {
				UserID: parseInt(cookie.getItem("id")),
				StripeToken: resp.id,
				CardholderName: resp.card.name,
				ExpMonth: resp.card.exp_month,
				ExpYear: resp.card.exp_year,
				Brand: _this.brand(),
				Last4: resp.card.last4,
				AddressZip: _this.zip5()
			};
			m.request({
				method: "POST",
				url: "/api/cards.json",
				data: data
			}).then(function(data) {
				deferred.resolve(data);
			}, function(err) {
				deferred.reject(new Error(err.Msg));
			});
		}, function(err) {
			deferred.reject(err);
		});
		return deferred.promise;
	};
	var saveStripe = function() {
		var deferred = m.deferred();
		Stripe.card.createToken({
			number: _this.number(),
			cvc: _this.cvc(),
			exp: _this.expiry(),
			name: _this.cardholderName(),
			address_zip: _this.zip5()
		}, function(status, response) {
			if (response.error) {
				return deferred.reject(new Error(response.error.message));
			}
			deferred.resolve(response);
		});
		return deferred.promise;
	};
};

Card.brandIcon = function(brand) {
	var icon;
	console.log("brand: " + brand);
	switch(brand) {
	case "American Express", "Diners", "Discover", "JCB", "Maestro",
		"MasterCard", "PayPal", "Visa":
		var imgPath = brand.toLowerCase().replace(" ", "_");
		imgPath = "card_" + imgPath + ".svg";
		imgPath = "/public/images/" + imgPath;
		icon = m("img", { src: imgPath, class: "icon-fit" });
		break;
	default:
		icon = m("span", brand);
		break;
	}
	return icon;
};
