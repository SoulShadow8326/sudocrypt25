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
			for(let i=1;i<=ctfCount;i++){ lvlLabels.push('ctf-'+i); lvlData.push(1); lvlColors.push('#36A2EB'); }
			for(let i=1;i<=crypticCount;i++){ lvlLabels.push('cryptic-'+i); lvlData.push(1); lvlColors.push('#9722e5'); }
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

