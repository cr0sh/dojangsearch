package main

const webcontent = `
<!DOCTYPE html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>무릉도장 전적 검색기</title>
	<link rel="stylesheet" href="/bulma.css">
</head>
<body>
	<script src="/jquery.js"></script>
	<script src="/json3.js"></script>
	<script>
$("document").ready(function() {
	$("#frm").submit(function(event) {
		event.preventDefault();
		$("#result").text("전적 검색 중...");
		$.ajax({
			type: "POST",
			url: "/getrank",
			data: JSON.stringify({"World": parseInt($("#server").val(), 10), "Type": 2, "Name": $("#username").val()}),
			dataType: "json", 
			contentType: "application/json",
			success: function(data) {
				if (!data.Ok) {
					$("#result").html("서버에 저장된 전적이 없습니다.<br>" +
					"전적 수집 기간: " + formatDate(new Date(data.Start * 1000)) + " ~ " +
				formatDate(new Date(data.End * 1000)));
					return false;
				}
				$("#result").html(createResult(data));
			},
			error: function() {
				$("#result").text("검색 중 오류가 발생했습니다.");
			}
		});
	});
});

function createResult(data) {
	return "[최고 기록]<br>" + brief(data.MRank) +
		"<br>[최근 기록]<br>" + brief(data.Rank) +
		"<br>[추가 정보]<br>" +
		"직업군: " + data.Rank.job + "<br>" + 
		"세부직업: " + data.Rank.detail_job + "<br><br>" +
		"전적 수집 기간: " + formatDate(new Date(data.Start * 1000)) + " ~ " +
			formatDate(new Date(data.End * 1000));
}

function brief(target) {
	var date = new Date(target.checkedtime * 1000);
	return "도달: " + target.floor + "<br>" +
		"소요 시간: " + target.duration + "<br>" +
		"달성 날짜: " + formatDate(date) + "<br>";
}

function formatDate(date) {
	return date.getFullYear() + "년 " + (date.getMonth() + 1) + "월 " + date.getDate() + "일";
}
	</script>
	<form action="" id="frm">
		<input type="text" name="username" id="username" placeholder="캐릭터 이름">
		<select name="server" id="server">
			<option value="1">리부트</option>
			<option value="12">리부트2</option>
		</select>
		<input type="submit" value="검색">
		<br>
	</form>
	<div id="result">
		탐색 결과는 여기에 표시됩니다. <br>
		[주의]<br>
		<font color="red">
			정확한 탐색을 보증하지 않습니다.(Beta)<br>
			전적 DB 시스템은 8시간마다 공식 홈페이지 랭킹을 수집하므로, 변경된 전적 반영에 최대 24+8시간 소요될 수 있습니다.<br>
			전적 DB 시스템은 닉네임만으로 플레이어를 구분하므로 닉네임 변경에 취약합니다.<br>
		</font>
		<font color="blue">달성 날짜는 최대 ±1일의 오차가 존재합니다.</font><br><br>
		알림: 최근 랭킹의 달성 날짜가 적절하지 않은 오류가 발견되어 수정되었습니다. 그러나 이미 수집된 랭킹 DB를 수정할 수는 없으므로 2018-02-24 이전의 '최근 랭킹' 달성 날짜는 주 단위로만 정확함을 인지해 주시기 바랍니다. 2018-03-11에 2018-03-01부터 다시 수집한 DB로 교체할 예정입니다. (기존 DB도 교체 전까지 계속 업데이트됩니다)
		알림: 일요일에 달성한 무릉 기록이 정상적으로 기록되지 않는 현상이 발견되어 수정 중입니다. 2018-03-01부터 수집한 새 DB는 이를 정상적으로 반영합니다. (교체 전 DB도 일부 기록을 복구하였습니다.)
	</div>
</body>
</html>
`
