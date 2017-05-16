var app=angular.module("myApp", []);
app.controller("scoreCtrl", function ScoreControl($scope,$http){

$scope.connectWS = function(){
    var s = gotalk.connection().on('open',function(){
        $scope.connectionStatus="Connected";
        $scope.messages = new Array;

    });
};

gotalk.handleNotification('stat', function(scores){
    var js = JSON.parse(scores);
    $scope.Player1Score = js.Player1Score ;
	$scope.Player2Score = js.Player2Score ;
	$scope.Player3Score = js.Player3Score ;
	$scope.Player4Score = js.Player4Score ;
	$scope.Match        = js.Match        ;
	$scope.TotalBalls   = js.TotalBalls   ;
	$scope.BallInPlay   = js.BallInPlay   ;
	$scope.Display1     = js.Display1     ;
	$scope.Display2     = js.Display2     ;
	$scope.Display3     = js.Display3     ;
	$scope.Display4     = js.Display4     ;
	$scope.Credits      = js.Credits      ;

});


gotalk.handleNotification('msg', function(logEvent){
    var js = JSON.parse(logEvent);

    $scope.messages.push(js);
});

$scope.clearEvents = function(){
 $scope.messages = new Array;
}


});