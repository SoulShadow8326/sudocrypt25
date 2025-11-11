const container = document.querySelector('.profile-stats-row');
const chartCryptic = parseFloat(container?.dataset.levelsCryptic || '0');
const chartCtf = parseFloat(container?.dataset.levelsCtf || '0');

(function(){
	const s = document.createElement('script');
	s.src = 'https://cdn.jsdelivr.net/npm/chart.js';
	s.onload = function(){
		const canvas = document.getElementById('scoreChart');
		if(!canvas) return;
		const ctx = canvas.getContext('2d');
		let data, labels, colors;
		if((!chartCtf || chartCtf === 0) && (!chartCryptic || chartCryptic === 0)){
			labels = ['No Data'];
			data = [1];
			colors = ['#bdbdbd'];
		} else {
			labels = ['CTF','Cryptic'];
			data = [chartCtf, chartCryptic];
			colors = ['#36A2EB', '#9722e5'];
		}
		new Chart(ctx, {
			type: 'pie',
			data: {
				labels: labels,
				datasets: [{ data: data, backgroundColor: colors }]
			},
			options: { plugins: { legend: { position: 'bottom' } } }
		});

		const timeCanvas = document.getElementById('scoreTimeChart');
		if (timeCanvas) {
			const rawTimes = container?.dataset.scoreTimes || '[]';
			const rawPoints = container?.dataset.scorePoints || '[]';
			let timesArr = [];
			let pointsArr = [];
			try { timesArr = JSON.parse(rawTimes); } catch (e) { timesArr = []; }
			try { pointsArr = JSON.parse(rawPoints); } catch (e) { pointsArr = []; }
			const tc = timeCanvas.getContext('2d');
			if (timesArr.length && pointsArr.length) {
				const labelsTime = timesArr.map(ts => new Date(ts * 1000).toLocaleString());
				new Chart(tc, {
					type: 'line',
					data: {
						labels: labelsTime,
						datasets: [{
							label: 'Score',
							data: pointsArr,
							borderColor: '#9722e5',
							backgroundColor: 'rgba(151,34,229,0.1)',
							fill: true,
							tension: 0.2
						}]
					},
					options: {
						scales: {
							x: { display: true, title: { display: true, text: 'Time' } },
							y: { beginAtZero: true, title: { display: true, text: 'Score' } }
						},
						plugins: { legend: { display: false } }
					}
				});
			} else {
				new Chart(tc, {
					type: 'line',
					data: { labels: ['No Data'], datasets: [{ label: 'Score', data: [0], borderColor: '#bdbdbd', backgroundColor: 'rgba(189,189,189,0.1)', fill: true }] },
					options: { plugins: { legend: { display: false } } }
				});
			}
		}

		const levelsCanvas = document.getElementById('levelsChart');
		if(levelsCanvas){
			const lc = levelsCanvas.getContext('2d');
			const crypticCount = Math.max(0, Math.floor(chartCryptic));
			const ctfCount = Math.max(0, Math.floor(chartCtf));
			const lvlLabels = [];
			const lvlData = [];
			const lvlColors = [];
			for(let i=1;i<=ctfCount;i++){ lvlLabels.push('ctf+'+i); lvlData.push(1); lvlColors.push('#36A2EB'); }
			for(let i=1;i<=crypticCount;i++){ lvlLabels.push('cryptic+'+i); lvlData.push(1); lvlColors.push('#9722e5'); }
			if(lvlData.length===0){
				new Chart(lc, { type:'doughnut', data:{ labels:['No Levels'], datasets:[{ data:[1], backgroundColor:['#bdbdbd'] }] }, options:{ plugins:{ legend:{ position:'bottom' } } } });
			} else {
				new Chart(lc, { type:'doughnut', data:{ labels: lvlLabels, datasets:[{ data: lvlData, backgroundColor: lvlColors }] }, options:{ plugins:{ legend:{ position:'bottom' } }, maintainAspectRatio:true } });
			}
		}
	};
	document.head.appendChild(s);
})();

const notyf = new Notyf();

function enterEditMode(){
	document.getElementById('bioViewMode').style.display='none';
	document.getElementById('bioEditMode').style.display='block';
	document.getElementById('bioInput').focus();
}

function cancelEdit(){
	document.getElementById('bioEditMode').style.display='none';
	document.getElementById('bioViewMode').style.display='block';
}

async function saveBio(){
	const bio = document.getElementById('bioInput').value.trim();
	const bioPublic = document.getElementById('bioPublicCheck').checked;
	try{
		const res = await fetch('/api/user/update_bio',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({bio,bio_public:bioPublic})});
		const data = await res.json();
		if(res.ok){
			notyf.success('Bio updated successfully!');
			const bioTextEl=document.getElementById('bioText');
			if(bio){bioTextEl.innerHTML=bio;}else{bioTextEl.innerHTML='<span class="empty-bio">No bio yet. Click edit to add one!</span>'}
			const noticeEl=document.querySelector('.bio-private-notice');
			if(bioPublic){noticeEl.textContent='Bio status: public'}else{noticeEl.textContent='Bio status: private'}
			cancelEdit();
		}else{notyf.error(data.error||'Failed to update bio')}
	}catch(err){notyf.error('Failed to update bio')}
}

window.enterEditMode = enterEditMode;
window.cancelEdit = cancelEdit;
window.saveBio = saveBio;

let pollingInterval = null;
let allLogs = [];

async function fetchAttemptLogs() {
    try {
        const response = await fetch('/api/attempt_logs', {
            method: 'GET',
            credentials: 'include'
        });
        if (!response.ok) {
            console.error('Failed to fetch logs:', response.status);
            return;
        }
        const result = await response.json();
        if (typeof result.data !== 'string') {
            console.error('Invalid data format');
            return;
        }
        parseLogs(result.data);
        applyFilters();
    } catch (error) {
        console.error('Error fetching attempt logs:', error);
    }
}

function formatTime(unixTimestamp) {
    const date = new Date(unixTimestamp * 1000);
    return date.toLocaleString();
}

function parseLogs(logsString) {
    allLogs = [];
    const trimmed = logsString.trim();
    if (!trimmed) {
        return;
    }
    const logs = trimmed.split('\n').filter(entry => entry.length > 0);
    logs.forEach((logEntry) => {
        const parts = logEntry.split('+');
        if (parts.length < 3) {
            console.warn('Invalid log format:', logEntry);
            return;
        }
        const timeStr = parts[parts.length - 1];
        const typpe = parts[parts.length - 2];
        const attempt = parts.slice(0, parts.length - 2).join('+');
        const unixTime = parseInt(timeStr, 10);
        if (isNaN(unixTime)) {
            console.warn('Invalid timestamp:', timeStr);
            return;
        }
        allLogs.push({
            time: unixTime,
            formattedTime: formatTime(unixTime),
            type: typpe,
            attempt: attempt
        });
    });
}

function applyFilters() {
    const filterInput = document.querySelector('.filter_input');
    const filterSelect = document.getElementById('choices');
    if (!filterInput || !filterSelect) {
        displayLogs(allLogs);
        return;
    }
    const searchTerm = filterInput.value.trim(); 
    const filterType = filterSelect.value;
    let filtered = [...allLogs];
    
    if (searchTerm) {
        const lowerSearchTerm = searchTerm.toLowerCase();
        filtered = filtered.filter(log => {
            switch(filterType) {
                case 'opt1': 
                    return log.formattedTime.toLowerCase().startsWith(lowerSearchTerm);
                case 'opt2': 
                    return log.type.toLowerCase().startsWith(lowerSearchTerm);
                case 'opt3': 
                    return log.attempt.toLowerCase().startsWith(lowerSearchTerm);
                default: //no point butok
                    return log.formattedTime.toLowerCase().startsWith(lowerSearchTerm) ||
                           log.type.toLowerCase().startsWith(lowerSearchTerm) ||
                           log.attempt.toLowerCase().startsWith(lowerSearchTerm);
            }
        });
    }
    displayLogs(filtered);
}

function displayLogs(logs) {
    const logsContainer = document.querySelector('.logs');
    if (!logsContainer) {
        console.error('Logs container not found');
        return;
    }
    logsContainer.innerHTML = '';
    if (logs.length === 0) {
        logsContainer.innerHTML = '<p class="empty-logs">No attempt logs found.</p>';
        return;
    }
    logs.forEach((log, index) => {
        const logDiv = document.createElement('div');
        logDiv.className = 'log';
        const timePara = document.createElement('p');
        timePara.textContent = `${index + 1}. ${log.formattedTime}`;
        const xPara = document.createElement('p');
        xPara.textContent = log.type;
        const attemptPara = document.createElement('p');
        attemptPara.className = 'log_el';
        attemptPara.textContent = log.attempt;
        logDiv.appendChild(timePara);
        logDiv.appendChild(xPara);
        logDiv.appendChild(attemptPara);
        logsContainer.appendChild(logDiv);
    });
}

function startPolling() {
    fetchAttemptLogs();
    pollingInterval = setInterval(() => {
        fetchAttemptLogs();
    }, 60000);
}

function stopPolling() {
    if (pollingInterval) {
        clearInterval(pollingInterval);
        pollingInterval = null;
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const logsContainer = document.querySelector('.attempt_logs_container');
    if (logsContainer) {
        startPolling();
        const filterInput = document.querySelector('.filter_input');
        const filterSelect = document.getElementById('choices');
        if (filterInput) {
            filterInput.addEventListener('input', applyFilters);
        }
        if (filterSelect) {
            filterSelect.addEventListener('change', applyFilters);
        }
    }
});

document.addEventListener('visibilitychange', () => {
    const logsContainer = document.querySelector('.attempt_logs_container');
    if (!logsContainer) return;
    if (document.hidden) {
        stopPolling();
    } else {
        startPolling();
    }
});

window.addEventListener('beforeunload', () => {
    stopPolling();
});

