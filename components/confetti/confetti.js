let W = window.innerWidth - 30;
let H = window.innerHeight - 30;
const canvas = document.getElementById("canvas");
const context = canvas.getContext("2d");
const maxConfettis = 300;
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
    this.active = false;
    this.x = 0;
    this.y = 0;
    this.vx = 0;
    this.vy = 0;
    this.r = 6;
    this.color = possibleColors[Math.floor(Math.random() * possibleColors.length)];
    this.tilt = 0;
    this.tiltAngleIncremental = Math.random() * 0.07 + 0.05;
    this.tiltAngle = 0;
    this.life = 0;
    this.maxLife = 0;
    this.burstToken = 0;

    this.resetForBurst = function (bx, by, token) {
        this.active = true;
        this.x = bx;
        this.y = by;
    const angle = Math.random() * Math.PI * 2;
    const speed = Math.random() * 8 + 4;
        this.vx = Math.cos(angle) * speed;
        this.vy = Math.sin(angle) * speed;
        this.r = randomFromTo(6, 18);
        this.color = possibleColors[Math.floor(Math.random() * possibleColors.length)];
        this.tilt = Math.random() * 10 - 5;
        this.tiltAngleIncremental = Math.random() * 0.2 + 0.05;
        this.tiltAngle = 0;
        this.life = 0;
        this.maxLife = randomFromTo(12, 18);
        this.burstToken = token || 0;
    };

    this.draw = function () {
        if (!this.active) return;
        context.beginPath();
        context.lineWidth = this.r / 2;
        context.strokeStyle = this.color;
        context.moveTo(this.x + this.tilt + this.r / 3, this.y);
        context.lineTo(this.x + this.tilt, this.y + this.tilt + this.r / 5);
        context.stroke();
    };
}

let lastBurst = 0;
let nextBurstIn = randomFromTo(10, 80);
let burstInProgress = false;
let activeBurstToken = 0;
let activeBurstRemaining = 0;
let burstTokenCounter = 1;

function createBurst() {
    if (burstInProgress) return 0;
    burstInProgress = true;
    const token = burstTokenCounter++;
    activeBurstToken = token;
    activeBurstRemaining = 0;
    const bx = Math.random() * W;
    const by = Math.random() * H * 0.7 + H * 0.1;
    const count = randomFromTo(10, 30);
    let created = 0;
    for (let i = 0; i < maxConfettis && created < count; i++) {
        if (!particles[i].active) {
            particles[i].resetForBurst(bx, by, token);
            created++;
            activeBurstRemaining++;
        }
    }
    for (let i = 0; i < maxConfettis && created < count; i++) {
        const idx = Math.floor(Math.random() * maxConfettis);
        particles[idx].resetForBurst(bx, by, token);
        created++;
        activeBurstRemaining++;
    }
    lastBurst = Date.now();
    nextBurstIn = randomFromTo(10, 120);
    return created;
}

function Draw() {
    requestAnimationFrame(Draw);
    context.clearRect(0, 0, W, window.innerHeight);
    const now = Date.now();
    if (now - lastBurst > nextBurstIn && !burstInProgress) {
        createBurst();
    }
    for (let i = 0; i < maxConfettis; i++) {
        const p = particles[i];
        if (!p.active) continue;
        p.draw();
        p.tiltAngle += p.tiltAngleIncremental;
    p.x += p.vx;
    p.y += p.vy;
    p.vy += 0.25;
        p.vx *= 0.995;
        p.vy *= 0.999;
        p.tilt = Math.sin(p.tiltAngle) * 15;
        p.life++;
        if (p.life > p.maxLife || p.y > H + 50 || p.x < -50 || p.x > W + 50) {
            const token = p.burstToken;
            p.active = false;
            p.burstToken = 0;
            if (token === activeBurstToken) {
                activeBurstRemaining--;
                if (activeBurstRemaining <= 0) burstInProgress = false;
            }
        }
    }
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
            document.querySelector(".site-footer").style.opacity = 0

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
							document.querySelector(".site-footer").style.opacity = 0
                            resolve();
                        }, Math.min(Math.max(window.innerHeight, 818), 882) * (6/10)) //trust.
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
