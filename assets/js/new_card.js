(function(ava) {
ava.NewCard = {}
ava.NewCard.controller = function(props) {
	if (!ava.isLoggedIn()) {
		return m.route("/login?r=" + encodeURIComponent(window.location.search))
	}
	var ctrl = this
	var saveBtn = function() {
		return document.getElementById("card-save-btn")
	}
	var cancelBtn = function() {
		return document.getElementById("card-cancel-btn")
	}
	var errorHolder = function() {
		return document.getElementById("card-error")
	}
	var cardNumberHolder = function() {
		return document.getElementById("card-number")
	}
	var cardExpiryHolder = function() {
		return document.getElementById("card-expiry")
	}
	var cardCVCHolder = function() {
		return document.getElementById("card-cvc")
	}
	ctrl.saveCard = function(ev) {
		ev.preventDefault()
		if (ctrl.props.saving()) {
			return
		}
		ctrl.save()
		ctrl.props.error(ctrl.vm.validateFields())
		if (!!ctrl.props.error()) {
			ctrl.saveComplete()
			return
		}
		ctrl.props.card().save().then(function(data) {
			m.route("/profile")
			m.redraw()
		}, function(err) {
			ctrl.props.error(err.message)
			ctrl.saveComplete()
		})
	}
	ctrl.saveAgain = function() {
		var deferred = m.deferred()
		ctrl.saveStripe().then(function(resp) {
			_this.brand(resp.card.brand)
			var data = {
				UserID: parseInt(cookie.getItem("id")),
				StripeToken: resp.id,
				CardholderName: resp.card.name,
				ExpMonth: resp.card.exp_month,
				ExpYear: resp.card.exp_year,
				Brand: _this.brand(),
				Last4: resp.card.last4,
				AddressZip: _this.zip5()
			}
			m.request({
				method: "POST",
				url: "/api/cards.json",
				data: data
			}).then(function(data) {
				deferred.resolve(data)
			}, function(err) {
				deferred.reject(new Error(err.Msg))
			})
		}, function(err) {
			deferred.reject(err)
		})
		return deferred.promise
	}
	ctrl.saveStripe = function() {
		var deferred = m.deferred()
		Stripe.card.createToken({
			number: _this.number(),
			cvc: _this.cvc(),
			exp: _this.expiry(),
			name: _this.cardholderName(),
			address_zip: _this.zip5()
		}, function(status, response) {
			if (response.error) {
				return deferred.reject(new Error(response.error.message))
			}
			deferred.resolve(response)
		})
		return deferred.promise
	}
	ctrl.save = function() {
		ctrl.props.saving(true)
		ctrl.props.savingText("Saving...")
		cancelBtn().classList.add("hidden")
		errorHolder().classList.add("hidden")
	}
	ctrl.saveComplete = function() {
		ctrl.props.saving(false)
		cancelBtn().classList.remove("hidden")
		ctrl.props.savingText("Save")
		if (!!ctrl.props.error()) {
			errorHolder().innerText = ctrl.props.error()
			errorHolder().classList.remove("hidden")
		} else {
			errorHolder().classList.add("hidden")
		}
	}
	ctrl.validateFields = function() {
		var card = ctrl.props.card
		if (Stripe.card.validateCardNumber(card.number())) {
			cardNumberHolder().classList.remove("has-error")
		} else {
			cardNumberHolder().classList.add("has-error")
			return "Card number is invalid."
		}
		if (Stripe.card.validateExpiry(card.expiry())) {
			cardExpiryHolder().classList.remove("has-error")
		} else {
			cardExpiryHolder().classList.add("has-error")
			return "Card expiration is invalid."
		}
		if (Stripe.card.validateCVC(card.cvc())) {
			cardCVCHolder().classList.remove("has-error")
		} else {
			cardCVCHolder().classList.add("has-error")
			return "Card CVC is invalid. This is the 3 or 4 digit security code."
		}
		return ""
	}
	props = props || {}
	ctrl.props = {
		error: m.prop(""),
		saving: m.prop(false),
		savingText: m.prop("Save"),
		card: {
			id: m.prop(props.id || 0),
			cardholderName: m.prop(props.cardholderName || ""),
			number: m.prop(props.number || ""),
			expMonth: m.prop(props.expMonth || ""),
			expYear: m.prop(props.expYear || ""),
			expiry: m.prop(props.expiry || ""),
			cvc: m.prop(props.cvc || ""),
			zip5: m.prop(props.zip5 || ""),
			brand: m.prop(""),
			last4: m.prop(props.last4 || "")
		}
	}
	if ((ctrl.props.card.expMonth()+ctrl.props.expYear()).length > 0) {
		if (ctrl.props.card.expiry().length === 0) {
			var x = ctrl.props.card.expMonth() + " / " + ctrl.props.expYear()
			ctrl.props.card.expiry(x)
		}
	}
}
ava.NewCard.view = function(ctrl) {
	return m(".body", [
		m.component(ava.Header),
		ava.NewCard.addView(ctrl),
		m.component(ava.Footer)
	])
}
ava.NewCard.addView = function(ctrl) {
	cformdata = [
		{field:"number", ph: "4444 0000 0000 1234", txt: "Card number"},
		{field:"expiry", ph: "01 / 2017", txt: "Expires"},
		{field:"cvc", ph: "123", txt: "CVC"},
		{field:"cardholderName", ph: "Cardholder name", txt: "Cardholder name"},
		{field:"zip5", ph: "90210", txt: "Billing Zip"},
	]
	var formfields = cformdata.map(function(fd) {
		return m("#card-"+fd.field+".form-group", [
			m("label.col-md-3.control-label", fd.txt),
			m(".col-md-9", [
				m("input", {
					class: "form-control",
					type: "text",
					placeholder: fd.ph,
					onchange: m.withAttr("value", ctrl.props.card[fd.field]),
					value: ctrl.props.card[fd.field]()
				})
			])
		])
	})
	return m("#full.container", [
		m(".row.margin-top-sm", m(".col-md-12", m("h1", "Add Card"))),
		m(".row margin-top-sm", [
			m("form.col-md-7.card", [
				m("div", {
					id: "card-error",
					class: "alert alert-danger hidden"
				}, ctrl.props.error()),
				m(".form-horizontal", formfields),
				m(".text-right", [
					m("a", {
						id: "card-cancel-btn",
						class: "btn btn-sm",
						href: "/profile",
						config: m.route
					}, "Cancel"),
					m("input", {
						id: "card-save-btn",
						type: "submit",
						class: "btn btn-primary btn-sm btn-collection",
						value: ctrl.props.savingText(),
						onclick: ctrl.saveCard,
						onsubmit: ctrl.saveCard
					})
				])
			])
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
