## БОЛЕЕ ТОЧНЫЕ ТАЙМЕРЫ, СРАБАТЫВАЮТ ЧАЩЕ, ЧЕМ 10 РАЗ В СЕКУНДУ

# t - время до события в секундах, например, .05 - это 50 миллисекунд
# action - экшен, что запускается по окончании ожидания
# loop=True - бесконечные повторы, loop=1 - одноразовое событие, loop=3 - трёхразовое
# id - уникальный идентификатор (если None, то создаётся автоматически)
# $ add_timer(t=1., action=Notify('DONE'), loop=1, id=None)

# для авто-id
default timer_id_counter = 0

# все игровые события будут заноситься в этот словарь
default game_timers = { }

init python:
    import datetime

    # события хранятся в виде кортежа из трёх позиций
    # (start_time, t, action, loop)
    # (время начала, длительность ожидания в секундах, функция, количество повторов)
    # добавление события в список, ожидание стартует сразу
    def add_timer(t=1., action=Notify('DONE'), loop=1, id=None):
        global timer_id_counter

        if id is None:
            id = "auto_id_%s" % timer_id_counter
            timer_id_counter += 1

        store.game_timers[id] = ((datetime.datetime.now() - datetime.datetime.fromtimestamp(0)).total_seconds(), t, action, int(loop))

    # удалить событие с защитой от дурака и с возможностью его сохранения в переменной
    def pop_timer(id):
        if id in game_timers.keys(): return store.game_timers.pop(id)

        return None

    # удалить таймеры, id которых начинаются с указанно1 строки 
    def del_timers_startswith(id):
        all = [ i for i in game_timers.keys() if i.startswith(id) ]

        for i in all: pop_timer(i)

    # вычислить, сколько осталось времени, исходя из
    # текущего времени, времени старта и длительности
    def get_time_left(start, t, subzero=False):
        now = (datetime.datetime.now() - datetime.datetime.fromtimestamp(0)).total_seconds()
        res = t - now + start

        if not subzero:
            if res < 0: res = 0

        return res

    # функция для проверки событий
    def timer_second_at_f(trans, st, at):
        global game_timers

        # список id таймеров для уничтожения
        kill = []

        # перебираем все события в списке
        for id in list(game_timers.keys()):
            # параметры события
            start, t, action, loop = game_timers[id]

            # текущее время в секундах
            now = (datetime.datetime.now() - datetime.datetime.fromtimestamp(0)).total_seconds()

            # проверяем, закончилось ли ожидание
            if now >= start + t:
                # если событие повторяющееся, то запускаем повтор
                if loop:
                    # это на случай ограниченного количества повторов
                    if isinstance(loop, int): loop -= 1

                    game_timers[id] = (now, t, action, loop)

                # если одноразовое или кончились повторы - удаляем его из словаря
                if not loop: kill.append(id)

                # если пора, то запускаем событие, к примеру, функция renpy.notify('Пора!')
                if action: action()

        # уничтожение ненужных таймеров
        for i in kill: pop_timer(i)

        return 1. / config.framerate

    # получить текущее значение обратного отсчёта для события id
    # subzero - допустимы ли отрицательные значения для вычисления пропущенных циклов
    # sec - округлить до секунд
    # format="{:02d}" - перевести в строку и добавить ноль спереди, если число меньше 10
    def time_left(id="timer", subzero=False, sec=False, format=False):
        res = 0

        if id in game_timers.keys():
            start, t = game_timers[id][0], game_timers[id][1]
            res = get_time_left(start, t, subzero)

            if sec: res = int(round(res))

        if format: res = format.format(res)

        return res

init python:
    # отдельный слой для таймеров, чтобы они не останавливались по HideInterface()
    renpy.add_layer("timers", above="master")

    # экран с таймерами показан, пока запущен скрипт игры
    config.overlay_screens.append("timerscreen")

# трансформ для постоянной проверки таймеров
transform timer_second_at():
    function renpy.curry(timer_second_at_f)()

# пустой пиксель
image timernull = Null(1, 1)

# экран для таймеров
screen timerscreen:
    layer "timers"

    # таймер, который постоянно вызывает функцию для проверки разных событий
    add "timernull" at timer_second_at
