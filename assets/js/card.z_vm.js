Card.vm = function(controller) {
	var saveBtn = function() {
		return document.getElementById("card-save-btn");
	};
	var cancelBtn = function() {
		return document.getElementById("card-cancel-btn");
	};
	var errorHolder = function() {
		return document.getElementById("card-error");
	};
	var cardNumberHolder = function() {
		return document.getElementById("card-number");
	}
	var cardExpiryHolder = function() {
		return document.getElementById("card-expiry");
	};
	var cardCVCHolder = function() {
		return document.getElementById("card-cvc");
	};
	var _this = this;
	_this.saving = m.prop(false);
	_this.savingText = m.prop("Save");
	_this.save = function() {
		_this.saving(true);
		_this.savingText("Saving...");
		cancelBtn().classList.add("hidden");
		errorHolder().classList.add("hidden");
	};
	_this.saveComplete = function() {
		_this.saving(false);
		cancelBtn().classList.remove("hidden");
		_this.savingText("Save");
		if (controller.error() != null && controller.error() !== "") {
			errorHolder().innerText = controller.error();
			errorHolder().classList.remove("hidden");
		} else {
			errorHolder().classList.add("hidden");
		}
	};
	_this.validateFields = function() {
		var card = controller.card;
		if (Stripe.card.validateCardNumber(card.number())) {
			cardNumberHolder().classList.remove("has-error");
		} else {
			cardNumberHolder().classList.add("has-error");
			return "Card number is invalid.";
		}
		if (Stripe.card.validateExpiry(card.expiry())) {
			cardExpiryHolder().classList.remove("has-error");
		} else {
			cardExpiryHolder().classList.add("has-error");
			return "Card expiration is invalid.";
		}
		if (Stripe.card.validateCVC(card.cvc())) {
			cardCVCHolder().classList.remove("has-error");
		} else {
			cardCVCHolder().classList.add("has-error");
			return "Card CVC is invalid. This is the 3 or 4 digit security code.";
		}
		return "";
	};
};
