let W = window.innerWidth - 30;
let H = window.innerHeight - 30;
const canvas = document.getElementById("canvas");
const context = canvas.getContext("2d");
const maxConfettis = 120;
const particles = [];

const possibleColors = [
    "DodgerBlue",
    "OliveDrab",
    "Gold",
    "Pink",
    "SlateBlue",
    "LightBlue",
    "Gold",
    "Violet",
    "PaleGreen",
    "SteelBlue",
    "SandyBrown",
    "Chocolate",
    "Crimson"
];

function randomFromTo(from, to) {
    return Math.floor(Math.random() * (to - from + 1) + from);
}

function confettiParticle() {
    this.x = Math.random() * W;
    this.y = Math.random() * H - H;
    this.r = randomFromTo(11, 33);
    this.d = Math.random() * maxConfettis + 11;
    this.color =
        possibleColors[Math.floor(Math.random() * possibleColors.length)];
    this.tilt = Math.floor(Math.random() * 33) - 11;
    this.tiltAngleIncremental = Math.random() * 0.07 + 0.05;
    this.tiltAngle = 0;

    this.draw = function () {
        context.beginPath();
        context.lineWidth = this.r / 2;
        context.strokeStyle = this.color;
        context.moveTo(this.x + this.tilt + this.r / 3, this.y);
        context.lineTo(this.x + this.tilt, this.y + this.tilt + this.r / 5);
        return context.stroke();
    };
}

function Draw() {
    const results = [];

    requestAnimationFrame(Draw);

    context.clearRect(0, 0, W, window.innerHeight);

    for (var i = 0; i < maxConfettis; i++) {
        results.push(particles[i].draw());
    }

    let particle = {};
    let remainingFlakes = 0;
    for (var i = 0; i < maxConfettis; i++) {
        particle = particles[i];

        particle.tiltAngle += particle.tiltAngleIncremental;
        particle.y += (Math.cos(particle.d) + 3 + particle.r / 2) / 2;
        particle.tilt = Math.sin(particle.tiltAngle - i / 3) * 15;

        if (particle.y <= H) remainingFlakes++;

        if (particle.x > W + 30 || particle.x < -30 || particle.y > H) {
            particle.x = Math.random() * W;
            particle.y = -30;
            particle.tilt = Math.floor(Math.random() * 10) - 20;
        }
    }

    return results;
}

window.addEventListener(
    "resize",
    function () {
        W = window.innerWidth;
        H = window.innerHeight;
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
    },
    false
);

for (var i = 0; i < maxConfettis; i++) {
    particles.push(new confettiParticle());
}

canvas.width = W;
canvas.height = H;


const showConfetti = getCookie("showConfetti")

document.addEventListener("DOMContentLoaded", () => {
    (async () => {
        if (showConfetti !== "false") {
            Draw();

            document.getElementById("confetti_div").style.display = "block"

            document.querySelector("#random div").style.display = "none"
            document.querySelector("#random div").style.position = "absolute"
            document.querySelector("#random div").style.top = window.innerHeight + 30 + "px"
            document.querySelector("#random div").style.opacity = 0

            new Promise(function (resolve) {
                setTimeout(() => {
                    document.getElementById("confetti_div").style.opacity = 0
                    document.querySelector("#random div").style.display = "block"
                    resolve();
                }, 3000)
            }).then(() => {
                new Promise(function (resolve) {
                    setTimeout(() => {
                        document.getElementById("confetti_div").style.display = "none"
                        document.querySelector("#random div").style.top = "0px"
                        document.querySelector("#random div").style.opacity = 1
                        resolve();
                    }, 300)
                }).then(() => {
                    new Promise(function (resolve) {
                        setTimeout(() => {
                            document.querySelector("#random div").style.position = "static";
                            setCookie("showConfetti", "false");
                            resolve();
                        }, 350)
                    })
                })
            })
        }
        else {
            document.getElementById("confetti_div").style.display = "none"
        }
    })()
})



function setCookie(name, value) {
    var date = new Date();
    date.setTime(date.getTime() + (24 * 60 * 60 * 1000));
    const expires = "; expires=" + date.toUTCString();
    document.cookie = name + "=" + (value || "") + expires + ";";
}
function getCookie(name) {
    var nameEQ = name + "=";
    var ca = document.cookie.split(';');
    for (var i = 0; i < ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) == ' ') c = c.substring(1, c.length);
        if (c.indexOf(nameEQ) == 0) return c.substring(nameEQ.length, c.length);
    }
    return null;
}