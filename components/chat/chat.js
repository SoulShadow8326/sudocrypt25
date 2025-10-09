function Signal(key, initialValue) {
	let value = initialValue;
	let onChange = null;
	return {
		Value: function () { return value; },
		setValue: function (newValue) { value = newValue; if (onChange) onChange(); },
		set onChange(callback) { onChange = callback; }
	};
}

var chatSignal = Signal('chatOpenState', 'close');

function setupChatSignalHandlers() {
	const chatToggleBtn = document.getElementById("chatToggleBtn");
	const chatPopup = document.getElementById("chatPopup");
	const chatCloseBtn = document.getElementById("chatCloseBtn");
	const chatMinimizeBtn = document.getElementById("chatMinimizeBtn");
	if (!chatToggleBtn || !chatPopup) {
		console.error('Chat elements not found:', { chatToggleBtn: !!chatToggleBtn, chatPopup: !!chatPopup });
		return;
	}
	chatSignal.onChange = function () {
		if (chatSignal.Value() === "open") {
			chatPopup.style.display = "flex";
			chatToggleBtn.style.opacity = 0;
			chatToggleBtn.style.transform = "scale(0)";
			setTimeout(function () {
				chatPopup.style.opacity = 1;
				chatPopup.style.transform = "translateY(0px)";
				chatToggleBtn.style.display = "none";
			}, 10);
			if (typeof refreshChatContent === 'function') refreshChatContent();
			const messagecontainer = document.getElementById("messagecontainer");
			setTimeout(function () {
				if (messagecontainer) messagecontainer.scrollTop = messagecontainer.scrollHeight;
			}, 200);
		} else {
			chatToggleBtn.style.display = "block";
			chatPopup.style.opacity = 0;
			chatPopup.style.transform = "translateY(900px)";
			setTimeout(function () {
				chatPopup.style.display = "none";
				chatToggleBtn.style.opacity = 1;
				chatToggleBtn.style.transform = "scale(1)";
			}, 400);
		}
	};
	if (chatToggleBtn) {
		chatToggleBtn.addEventListener("click", function () {
			chatSignal.setValue("open");
		});
	}
	if (chatCloseBtn) {
		chatCloseBtn.addEventListener("click", function () {
			chatSignal.setValue("close");
		});
	}
	if (chatMinimizeBtn) {
		chatMinimizeBtn.addEventListener("click", function () {
			chatSignal.setValue("close");
		});
	}
}

document.addEventListener('DOMContentLoaded', function () { setupChatSignalHandlers(); });


