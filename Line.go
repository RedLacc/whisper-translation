package main

type Line struct {
	Original   string //[00:02:02.440 --> 00:02:05.440]   Lando Norris leads the British Grand Prix.
	Start      string //00:02:02.440
	End        string //00:02:05.440
	Text       string //Lando Norris leads the British Grand Prix.
	Translated string //兰多诺里斯领跑英国大奖赛
}

func (l *Line) String() string {
	return l.Original + " => " + l.Translated
}
