# в кнопках всегда добавлять "[I]текст кнопки":
# чтобы они отличались потом от обычного текста
define I = "{#c}"

# открытые сектора на карте
default world_map = [ [ False for x in range(5) ] for y in range(5) ]

# события { label: ( icon, xpos, ypos, available ), ... }
default vevents = [ [ { } for x in range(5) ] for y in range(5) ]

# пустой слот сектора
image vnull = Null(246, 165)

style nope is empty:
    modal True
    xfill True
    yfill True

transform alpha_at(a=.5):
    alpha a

# кнопки для выбора доступного сектора
screen map_map(items):
    modal True

    frame:
        style "nope"
        at show_hide

        add "deskbg" align (.5, .5)
        add "map map" align (.5, .5)

        vbox:
            align (.5, .5)

            for y in range(5):
                hbox:
                    align (.5, .5)

                    for x in range(5):
                        if world_map[y][x]:
                            imagebutton idle "vnull" hover "map hover" action SetVariable("map_ex", x), SetVariable("map_ey", y), items[0].action

                        else:
                            add "map insensitive"

        # закрыть экран с глобальной картой
        button:
            style "ev_icon"
            add "icon close" align (.5, .5)
            pos (1., .0)
            offset (-65, 65)
            at hover_at
            action SetVariable("mode", "x"), items[-1].action

# стиль для иконок событий
style ev_icon is button:
    background "map button idle"
    hover_background "map button hover"
    selected_hover_background "map button hover"
    selected_background "map button selected"
    insensitive_background "map button insensitive"
    xysize (100, 100)
    padding (0, 0)

default ev_label = None

default vhistory = [ [ ] ]
default vhistory_height = 0
default book_page_shown = 0

init python:
    # перелистнуть на следующую страницу
    def page_flip_next():
        if book_page_shown + 2 < len(vhistory):
            store.book_state = "next"
            splay("page_flip")
            renpy.restart_interaction()

    PageFlipNext = renpy.curry(page_flip_next)

    # перелистнуть на предыдущую страницу
    def page_flip_prev():
        if book_page_shown - 2 >= 0:
            store.book_state = "prev"
            splay("page_flip")
            renpy.restart_interaction()

    PageFlipPrev = renpy.curry(page_flip_prev)

    # если места на странице не хватает, перелистываем на следующую
    def book_check(t=0, book=False):
        global vhistory, vhistory_height, book_state

        if isinstance(t, int):
            h = t

        else:
            # размер абзаца
            w, h = renpy.render(Text(t, style="book_text", xsize=book_w), config.screen_width, config.screen_height, 0, 0).get_size()
            h = int(h)

        # если абзац не помещается в страницу, создаём новую
        if vhistory_height + h >= (book_h - book_padding[1] * 2 - book_h1 - 140):
            vhistory_height = 0
            vhistory.append([ ])

            # перелистываем на новую страницу
            if len(vhistory) % 2:
                page_flip_next()

        # вычисляем новую высоту текста на странице
        vhistory_height += h

    # добавить текущий диалог в историю диалогов для отображения на экране
    def book_say(who=None, what=None):
        global vhistory, book_state

        t = what

        if who: t = "[I]{b}" + who + ": {/b}\n" + t

        # проверяем, хватает ли места
        book_check(t)

        # добавляем строки
        vhistory[len(vhistory) - 1].append((who, what))

        # обновляем экран
        renpy.hide_screen("book")
        renpy.show_screen("book", att=only_hide)
        renpy.restart_interaction()

    BookSay = renpy.curry(book_say)

    def event_start(label):
        store.mode = "call"
        store.ev_label = label

    EventStart = renpy.curry(event_start)

default map_ex = 0
default map_ey = 0

# карта событий в указанном секторе
screen map_events(items):
    frame:
        style "nope"
        at show_hide

        add "deskbg" align (.5, .5)

        # карта сектора
        $ bg = "map ev %s%s" % ("abcde"[map_ey], map_ex + 1)

        if not has_image(bg):
            $ bg = "map ev none"

        add bg align (.5, .5)

        # события сектора
        for i in vevents[map_ey][map_ex].keys():
            $ label = str(i)
            $ icon, xpos, ypos, available = vevents[map_ey][map_ex][label]

            button:
                style "ev_icon"
                add "map icon " + icon align (.5, .5)
                pos (xpos, ypos)
                sensitive available
                action EventStart(label), items[0].action

                if available:
                    at hover_at

                else:
                    at alpha_at

        # закрыть карту событий сектора
        button:
            style "ev_icon"
            add "icon close" align (.5, .5)
            pos (1., .0)
            offset (-65, 65)
            at hover_at
            action SetVariable("mode", "events"), items[-1].action

# появление со сдвигом
transform f_in(t=.2, dx=-250, dy=0):
    subpixel True
    offset (dx, dy)
    alpha 0
    easein t offset (0, 0) alpha 1

transform zoom_at(z=1):
    subpixel True
    zoom z

# размеры страницы
define book_w = 660
define book_h = 880

# отступы внутри контейнера страницы
define book_padding = (40, 40)

# контейнер страницы
style book_frame is empty:
    yalign .5
    xysize (book_w, book_h)
    padding book_padding

# контейнер для строк текста на странице
style book_vbox is vbox:
    xsize book_w - book_padding[0] * 2
    xmaximum book_w - book_padding[0] * 2
    xminimum book_w - book_padding[0] * 2

style book_hbox is hbox:
    xsize book_w - book_padding[0] * 2
    xmaximum book_w - book_padding[0] * 2
    xminimum book_w - book_padding[0] * 2

# текст в книге
style book_text is text:
    font "font/text.otf"
    size 28
    color "#221"
    justify True
    padding (0, 0)
    margin (0, 0)
    text_align 0
    xalign 0
    italic False # Курсив включен True

# центрированный текст
style book_text_c is book_text:
    layout "subtitle"
    text_align .5
    xalign .5

# текст справа
style book_text_r is book_text_c:
    text_align 1.
    xalign 1.

# заменим экран say
screen say(who, what):
    if book():
        timer .0001 action BookSay(who, what)

        if who:
            text who id "who" at zoom_at(0)

        text what id "what" at zoom_at(0)

    else:
        # ВАЖНО:
        # Убираем on "hide" action SayHide()
        # Он срабатывал между репликами и заставлял namebox моргать.

        window:
            id "window"

            if who:
                window:
                    at who_at(who), eval(who_at_s)
                    id "namebox"
                    style "namebox"

                    hbox:
                        text who id "who"

            text what id "what"

        add SideImage() xalign 0.0 yalign 1.0

# состояние перелистывания книги - на месте, назад, вперёд ("", "prev", "next")
default book_state = ""

define book_t = .2
define book_b = -.5

# левая не перелистывается
transform book_l2l():
    xzoom 1
    anchor (1., .5)
    pos (.5, .5)
    matrixcolor BrightnessMatrix(0)

# правая не перелистывается
transform book_r2r():
    xzoom 1
    anchor (.0, .5)
    pos (.5, .5)
    matrixcolor BrightnessMatrix(0)

# перелистывается слева в центр
transform book_l2c(t=book_t, b=book_b):
    subpixel True
    xzoom 1
    anchor (1., .5)
    pos (.5, .5)
    matrixcolor BrightnessMatrix(0)
    ease t xzoom 0 matrixcolor BrightnessMatrix(b)

# перелистывается из центра вправо
transform book_c2r(t=book_t, b=book_b):
    subpixel True
    xzoom 0
    anchor (.0, .5)
    pos (.5, .5)
    matrixcolor BrightnessMatrix(b)
    t
    ease t xzoom 1 matrixcolor BrightnessMatrix(0)

# перелистывается справа в центр
transform book_r2c(t=book_t, b=book_b):
    subpixel True
    xzoom 1
    anchor (.0, .5)
    pos (.5, .5)
    matrixcolor BrightnessMatrix(0)
    ease t xzoom 0 matrixcolor BrightnessMatrix(b)

# перелистывается из центра влево
transform book_c2l(t=book_t, b=book_b):
    subpixel True
    xzoom 0
    anchor (1., .5)
    pos (.5, .5)
    matrixcolor BrightnessMatrix(b)
    t
    ease t xzoom 1 matrixcolor BrightnessMatrix(0)

# какие страницы куда листать
define book_states = {
    "": [ (0, book_l2l, "L"), (1, book_r2r, "R") ],
    "prev": [ (-2, book_l2l, "L"), (1, book_r2r, "R"), (-1, book_c2r, "R"), (0, book_l2c, "L") ],
    "next": [ (0, book_l2l, "L"), (3, book_r2r, "R"), (2, book_c2l, "L"), (1, book_r2c, "R") ],
}

init python:
    # заканчиваем перелистывать страницу
    def book_flip_end():
        global book_page_shown, book_state

        if book_state == "prev":
            if book_page_shown - 2 >= 0: book_page_shown -= 2

        elif book_state == "next":
            if book_page_shown + 2 < len(vhistory): book_page_shown += 2

        book_state = ""
        renpy.restart_interaction()

    BookFlipEnd = renpy.curry(book_flip_end)

    # есть тег центрирования?
    def is_c(s):
        return ("{#c}" in s) or ("[I]" in s)

    # режим книги?
    def book():
        return mode in ( "book", "end" )

    # для центрирования
    def get_style(s):
        if is_c(s): return "book_text_c"

        if "{#r}" in s: return "book_text_r"

        return "book_text"

    # вычесть ресурсы
    def res_dec(i):
        t = i.kwargs.get("vif", None)

        if t:
            vdec(t[0], t[1])
            splay(t[0])

    ResDec = renpy.curry(res_dec)

transform crop(x, y, w, h):
    crop(x, y, w, h)

# страница книги
screen book_page(plus=0, at=None, side=None, z=0):
    zorder z

    $ index = book_page_shown + plus
    $ page = [ ]
    $ xalign = 1. if side == "L" else 0

    if 0 <= index < len(vhistory):
        $ page = vhistory[index]

    frame:
        style "empty"
        xysize (config.screen_width // 2, config.screen_height)
        background "book page" + side
        at at

        style_prefix "book"

        frame:
            xalign xalign

            vbox:
                xalign xalign

                for i in range(len(page)):
                    $ vwho, vwhat = page[i]
                    $ w = "" if is_c(vwhat) else "   "

                    if vwho:
                        text "%s%s" % (vwho, ":") style get_style(vwho) bold True

                    text "%s%s" % (w, vwhat) style get_style(vwhat):
                        if index == len(vhistory) - 1:
                            if vhistory[index]:
                                if i == len(vhistory[index]) - 1:
                                    slow_cps preferences.text_cps

                if book_items and index == len(vhistory) - 1:
                    null height 24

                    for i in book_items:
                        $ s, iif = vget_siif(**i.kwargs)

                        button:
                            if iif:
                                # "ВЫБОР" переводим отдельно, без служебного тега {#c}.
                                # Иначе Ren'Py ищет перевод строки "{#c}\nВЫБОР",
                                # а в переводах обычно есть простое "ВЫБОР".
                                action ResDec(i), SetVariable("book_items", [ ]), Hide("choice"), BookSay("{#c}\n" + renpy.translate_string("ВЫБОР"), renpy.translate_string(i.caption)), i.action, StopRoll()

                            else:
                                at alpha_at
                                action [ ]

                            hbox style "hbox":
                                xalign .5
                                spacing 24

                                text i.caption style "book_button_text"
                                # slow_cps preferences.text_cps

                                if s:
                                    text s style "book_button_text"
                                    # slow_cps preferences.text_cps

            # кнопка для завершения
            if mode == "end" and index == len(vhistory) - 1:
                textbutton _("ЗАВЕРШИТЬ") style "book_end" action SetVariable("mode", ""), End() yoffset 20

            else:
                # номер страницы
                text str(index + 2) style "book_text" align (.5, 1.)

            # кнопки для перелистывания
            if book():
                if side == "R" and index < len(vhistory) - 1:
                    imagebutton idle "book pageR idle" hover "book pageR hover" align (1., 1.) action PageFlipNext() yoffset 20

                elif side == "L" and index > 1:
                    imagebutton idle "book pageL idle" hover "book pageL hover" align (.0, 1.) action PageFlipPrev() yoffset 20

# кнопки в книге
style book_button is choice_button:
    xsize book_w - book_padding[0] * 2
    yminimum 70
    xalign .5
    padding (100, 5)
    background Frame("book button idle", 70, 0)
    hover_background Frame("book button hover", 70, 0)

style book_button_text is choice_button_text:
    font "font/text.otf"
    size 24
    color "#554"
    hover_color "#332"
    insensitive_color "#5548"
    text_align .5
    layout "subtitle"
    align (.5, .5)
    yoffset 12
    justify False

# style choice_button_text:
    # align (.5, .5)
    # text_align .5
    # layout "subtitle"

init python:
    # появление экрана выбора - поместятся ли кнопки на странице книги
    def choice(items):
        global book_items, vhistory_height, vhistory, book_state

        if book():
            book_items = items
            h = 24 + 70 * len(items)

            # если абзац не помещается в страницу, создаём новую
            if vhistory_height + h >= (book_h - book_padding[1] * 2 - book_h1 - 70):
                vhistory_height = 0
                vhistory.append([ ])

                # перелистываем на новую страницу
                if len(vhistory) % 2:
                    book_state = "next"
                    splay("page_flip")

            # обновляем экран
            renpy.restart_interaction()

    Choice = renpy.curry(choice)

    # def book_get_pos():
        # d = renpy.get_displayable(screen="book", id="book_choice", layer="screens")
        # xy = renpy.get_placement(d)

        # return (xy.xpos, xy.ypos)

    # завершить паузу
    def end():
        Hide("quick_line")()
        Show("quick_line")()
        renpy.end_interaction(True)

    End = renpy.curry(end)

    # клик по книге
    def book_click():
        # если нет выборов
        if not book_items:
            # если не последний разворот
            if book_page_shown < len(vhistory) - 2:
                # листаем вперёд
                page_flip_next()

            # если последний разворот
            elif mode != "end":
                # следующий абзац
                end()

    BookClick = renpy.curry(book_click)

    import re

    # список тегов
    def vget_tags(text):
        return re.findall(r'\{([^=}]+)=([^}]+)\}', text)

    # получить значение последнего указанного тега
    def vget_tag(text, tag="#if"):
        res = [ i[1] for i in vget_tags(text) if i[0] == tag ]

        return (res[0] if res else "True")

    # хватает ли требуемого в ресурсах
    def vif(**kwargs):
        has = True

        for k in kwargs.keys():
            if k in sources.keys(): has &= sources[k] >= kwargs[k]

        return has

    # добавляем сколько-то чего-то в ресурсы
    def vinc(**kwargs):
        global sources

        for k in kwargs.keys():
            if k in sources.keys(): sources[k] = sources.get(k, 0) + kwargs[k]

    VInc = renpy.curry(vinc)

    # убираем что-то из ресурсов (если всего хватает)
    def vdec(**kwargs):
        global sources

        if vif(**kwargs):
            for k in kwargs.keys():
                if k in sources.keys(): sources[k] -= kwargs[k]

    VDec = renpy.curry(vdec)

    # строка для отображения количества требуемых для нажатия ресурсов
    def vneed(**kwargs):
        res = " ".join(list([ _("%s{image=mini %s}") % (kwargs[k], k) for k in kwargs.keys() if k in sources.keys() ]))
        add = kwargs.get("add", "")

        if res and add: res = res + " "

        return res + add

    # получить приписку к кнопке и условие
    def vget_siif(**kwargs):
        iif = kwargs.get("if", True)
        t = [ k for k in kwargs.keys() if k in sources.keys() ]

        if t: return vneed(**kwargs), vif(**kwargs) and iif

        return "", iif

    # всего ресурсов в наличии
    def A():
        return _("В наличии: ") + " ".join(list([ _("%s{image=mini %s}") % (sources.get(k, 0), k) for k in sources.keys() ]))

    # блокировать откат
    def stop_roll_ok():
        if stop_roll_flag:
            store.stop_roll_flag = False
            renpy.block_rollback()

    StopRollOK = renpy.curry(stop_roll_ok)

    # обозначить необходимость блокировки откатов
    def stop_roll():
        store.stop_roll_flag = True

    StopRoll = renpy.curry(stop_roll)

    config.overlay_screens.append("stop_roll_check")

# для автоматической блокировки выборов
screen stop_roll_check():
    timer .1 repeat True action StopRollOK()

default stop_roll_flag = False

# ресурсы персонажа (не инвентарь)
default sources = { "blood": 0, "money": 0 }

# копия кнопок выбора для встраивания в книгу
default book_items = [ ]

# заменим экран выборов
# при menu(m=1) все кнопки не текстовые, а картинками
screen choice(items, m=False):
    if m:
        for i in items:
            if not i.caption in pnc_deleted:
                imagebutton idle "bg item %s" % i.caption focus_mask True anchor (.5, .5) pos pnc_get_pos(i) at pnc_get_at(i) action Hide("sofagirl"), i.action

    else:
        timer .0001 action Choice(items)

        vbox:
            at show_hide

            style_prefix ( "book" if book() else "choice" )

            if book():
                yoffset config.screen_height

            for i in items:
                $ s, iif = vget_siif(**i.kwargs)

                button:
                    if iif and not book():
                        action i.action, StopRoll()

                    else:
                        at alpha_at
                        action [ ]

                    hbox style "hbox":
                        align (.5, .5)
                        spacing 24

                        text i.caption

                        if s:
                            text s

# свой текстбокс справа (+ заголовок и картинка слева вверху)
screen book(att=only_show):
    key [ "skip", "toggle_skip", "fast_skip" ] action NullAction()

    if book():
        # либо перелистываем, либо выводим следующий абзац
        key "dismiss" action BookClick()

    # конец перелистывания
    if book_state:
        timer book_t * 2 repeat True action BookFlipEnd()

    # фон книги
    frame:
        style "empty"
        add "deskbg"
        add "book bg"

        if att:
            at att

    # страницы книги
    for z in range(len(book_states[book_state])):
        $ plus, at, side = book_states[book_state][z]

        use book_page(plus, at, side, z)

default narrator_temp = Character(None)
default config_window_show_transition = config.window_show_transition
default config_window_hide_transition = config.window_hide_transition
# default rollback_temp = _rollback

# кнопка закрытия книги по завершении события
style book_end is button:
    xysize (300, 70)
    align (.5, 1.)
    padding (36, 12)
    background "book end idle"
    hover_background "book end hover"

# текст кнопки
style book_end_text is button_text:
    align (.5, .5)
    color "#554"
    hover_color "#992B2B"

default book_h1 = 48

## ВЫЗОВ СОБЫТИЯ С КАРТЫ СЕКТОРА В ВИДЕ ИСТОРИИ В КНИГЕ
label call_event:
    $ Preference("auto-forward", "disable")()
    $ skipping_temp = _skipping
    $ _skipping = False
    hide screen quick_line
    show screen quick_line

    python:
        # высота строки для номера страницы
        book_h1, temp = Text("1", style="book_text").size()
        book_h1 = int(book_h1)

        # включаем режим книги
        mode = "book"
        config_window_show_transition = config.window_show_transition
        config_window_hide_transition = config.window_hide_transition
        config.window_show_transition = None
        config.window_hide_transition = None
        # rollback_temp = _rollback
        # _rollback = False

        # очистить книгу
        vhistory = [ [ ] ]
        vhistory_height = 0
        book_page_shown = 0

    window show
    show screen book

    # вызываем сценарий
    call expression ev_label from _call_expression_ev_label

    # добавляем кнопку "завершить"
    $ mode = "end"

    # удаление последней пустой страницы
    if not vhistory[-1]:
        $ vhistory.pop()

    # ждём её нажатия
    while mode == "end":
        pause

    hide screen quick_line
    show screen quick_line

    # выключаем режим книги
    $ ev_label = ""
    hide screen book
    window hide
    $ config.window_show_transition = config_window_show_transition
    $ config.window_hide_transition = config_window_hide_transition
    $ mode = "events"

    $ _skipping = skipping_temp

    return

# Вызов боя из книжного события.
# Можно использовать в событиях книги так:
# call battle_from_book(allow_action=[ 0, 3 ])
#
# ВАЖНО:
# Основная защита теперь находится прямо в label battle.
# Поэтому бой будет нормально прятать/возвращать книгу даже если старый код
# всё ещё вызывает его через renpy.call_in_new_context("battle", ...).
label battle_from_book(bg="bg battle room", allow_attack=None, allow_action=None):
    call battle(bg=bg, allow_attack=allow_attack, allow_action=allow_action) from _call_battle_from_book

    return

