package notifier

type multiNotifier struct {
	notifiers []Notifier
}

// Multi creates a notifier that will forward the calls to the notifiers.
func Multi(notifiers ...Notifier) Notifier {
	if len(notifiers) == 1 {
		return notifiers[0]
	}
	n := make([]Notifier, len(notifiers))
	copy(n, notifiers)
	return &multiNotifier{n}
}

// BeforeTest implements the Notifier interface.
func (t *multiNotifier) Notify(r *Report) {
	for _, n := range t.notifiers {
		n.Notify(r)
	}
}
