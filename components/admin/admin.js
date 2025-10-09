const converter = new showdown.Converter();
const inputEl = document.getElementById('inputField');
const displayEl = document.getElementById('displayBox');
const popupContainer = document.getElementById('popupContainer');
const popupContent = document.getElementById('popupContent');
const sourceHintField = document.getElementById('sourceHintField');
const answerField = document.getElementById('answerField');
const levelName = document.getElementById("level_id")

function updateDisplay() {
    const inputValue = inputEl.value.trim();
    const html = converter.makeHtml(inputValue);
    displayEl.innerHTML = `${html}`;
    document.querySelectorAll("#displayBox a").forEach((x)=>{
        x.target="_blank"
    })
}

inputEl.addEventListener('input', updateDisplay);

popupContainer.addEventListener('click', function (e) {
    if (e.target === popupContainer) {
        closePopup()
    }
});

document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') {
        closePopup()
    }
});

function openPopup(levelNumber) {
    if (levelNumber == -1) {
        inputEl.value = ""
        displayEl.value = ""
        sourceHintField.value = ''
        answerField.value = ''
        popupContainer.style.display = 'flex'
        levelName.innerText = "New Level"
        document.getElementById("levelId").value = ""
        updateDisplay()
    } else {
        sourceHintField.value = levelsData[levelNumber]["sourcehint"]
        answerField.value = levelsData[levelNumber]["answer"]
        inputEl.value = levelsData[levelNumber]["markup"]
        popupContainer.style.display = 'flex'
        levelName.innerText = "Level " + String(levelNumber)
        document.getElementById("levelId").value = String(levelNumber)
        updateDisplay()
    }
}

function closePopup() {
    popupContainer.style.display = 'none';
}

function submitForm() {
    const sourceHint = sourceHintField.value.trim();
    const answer = answerField.value.trim();
    var levelId = document.getElementById("levelId").value.trim()
    if (/^[0-9]+$/.test(levelId)) {
        levelId = "cryptic-" + levelId
    }
    if (levelId === "") {
        levelId = "cryptic-0"
    }
    fetch("/set_level?source=" + encodeURIComponent(sourceHint) + "&answer=" + encodeURIComponent(answer) + "&markup=" + encodeURIComponent(inputEl.value.trim()) + "&levelid=" + String(levelId)).then((x) => {
        window.location = "/admin"
    })
}

function deleteLevel() {
    var levelId = document.getElementById("levelId").value.trim()
    if (/^[0-9]+$/.test(levelId)) {
        levelId = "cryptic-" + levelId
    }
    if (levelId === "") {
        levelId = "cryptic-0"
    }
    fetch("/delete_level?level=" + encodeURIComponent(levelId)).then(() => {
        window.location = "/admin"
    })
}