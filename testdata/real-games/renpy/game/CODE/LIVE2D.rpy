define l2d_tag = [ "gg", "ggcow" ]

# словарь со всеми эмоциями, разбитый на списки взаимоисключающих тегов
define gg_extras = [ {
    "emotions": [
        ( "default", "Стандарт" ),
        ( "anger", "Злость 1" ),
        ( "anger_2", "Злость 2" ),
        # ( "anger_3", "Злость 3" ),
        ( "anger_big", "Злость сильная 1" ),
        ( "anger_big_2", "Злость сильная 2" ),
        # ( "anger_big_3", "Злость сильная 3" ),
        ( "approval", "Одобрение 1" ),
        ( "approval_2", "Одобрение 2" ),
        # ( "approval_3", "Одобрение 3" ),
        ( "ded_inside", "Дединсайд 1" ),
        ( "ded_inside_2", "Дединсайд 2" ),
        # ( "ded_inside_3", "Дединсайд 3" ),
        ( "default", "По умолчанию" ),
        ( "embarrassment", "Смущение" ),
        # ( "embarrassment_3", "Смущение 3" ),
        ( "embarrassment_big", "Смущение сильное 1" ),
        ( "embarrassment_big_2", "Смущение сильное 2" ),
        # ( "embarrassment_big_3", "Смущение сильное 3" ),
        ( "excitement", "Возбуждение" ),
        ( "excitement_steam", "Возбуждение 3" ),
        ( "happy", "Счастье 1" ),
        ( "happy_2", "Счастье 2" ),
        # ( "happy_3", "Счастье 3" ),
        ( "happy_f", "Счастье 4" ),
        ( "love", "Любовь" ),
        # ( "love_3", "Любовь 3" ),
        ( "misunderstanding", "Непонимание 1" ),
        ( "misunderstanding_2", "Непонимание 2" ),
        # ( "misunderstanding_3", "Непонимание 3" ),
        ( "normal", "Обычное" ),
        ( "playfulness", "Игривость 1" ),
        ( "playfulness_2", "Игривость 2" ),
        # ( "playfulness_3", "Игривость 3" ),
        ( "resentment", "Негодование 1" ),
        ( "resentment_2", "Негодование 2" ),
        # ( "resentment_3", "Негодование 3" ),
        ( "shock", "Шок 1" ),
        ( "shock_2", "Шок 2" ),
        # ( "shock_3", "Шок 3" ),
        ( "shock_big", "Шок сильный 1" ),
        ( "shock_big_2", "Шок сильный 2" ),
        # ( "shock_big_3", "Шок сильный 3" ),
        ( "thoughtfulness", "Вдумчивость 1" ),
        ( "thoughtfulness_2", "Вдумчивость 2" ),
        # ( "thoughtfulness_3", "Вдумчивость 3" ),
    ],

    "mouth": [
        ( "open_mouth", "Открытый рот" ),
        ( "close_mouth", "Закрытый рот" ),
        ( "talk", "Рот говорит" ),
        ( "mute", "Рот молчит" ),
        ( "", "Без тега" )
    ],

    "clothes": [
        ( "casual_dress", "Платье - повседневное" ),
        ( "casual_blood_dress", "Платье - в крови" ),
        ( "dark_dress", "Тёмное платье" ),
        ( "naked_yes", "Бельё" ),
        ( "naked_dark_yes", "Тёмное бельё" ),
        ( "", "Без тега" )
    ],

    "bra": [
        ( "bra_yes", "Лифчик - да" ),
        ( "bra_no", "Лифчик - нет" ),
        ( "", "Без тега" )
    ],

    "pantsu": [
        ( "pantsu_yes", "Трусики - да" ),
        ( "pantsu_no", "Трусики - нет" ),
        ( "", "Без тега" )
    ],

    "hand": [
        ( "hand_1", "Рука 1" ),
        ( "hand_2", "Рука 2" ),
        ( "hand_3", "Рука 3" ),
        ( "hand_4", "Рука 4" ),
        ( "hand_5", "Рука 5" ),
        ( "", "Без тега")
    ],

    "eyes": [
        ( "eyes_open", "Глаза открыты" ),
        ( "eyes_closed", "Глаза закрыты" ),
        ( "", "Без тега" )
    ],

    "eye_left": [
        ( "eye_l_closed", "Левый глаз закрыт" ),
        ( "", "Без тега" )
    ],

    "eye_right": [
        ( "eye_r_closed", "Правый глаз закрыт" ),
        ( "", "Без тега" )
    ],

    "eyesx": [
        ( "zrachok_x_left", "Зрачок X - влево" ),
        ( "zrachok_x_center", "Зрачок X - центр" ),
        ( "zrachok_x_right", "Зрачок X - вправо" ),
        ( "", "Без тега" )
    ],

    "eyesy": [
        ( "zrachok_y_up", "Зрачок Y - вверх" ),
        ( "zrachok_y_center", "Зрачок Y - центр" ),
        ( "zrachok_y_down", "Зрачок Y - вниз" ),
        ( "", "Без тега")
    ],

    "steam": [
        ( "steam_yes", "Пар есть" ),
        ( "steam_no", "Пара нет" ),
        ( "", "Без тега")
    ],

    "wind": [
        ( "wind_speed_low", "Ветер - слабый" ),
        ( "wind_speed_middle", "Ветер - средний" ),
        ( "wind_speed_high", "Ветер - сильный" ),
        ( "", "Без тега" )
    ]
}, {
    "all": [
        ( "scene1", "1" ),
        ( "scene2a", "2A" ),
        ( "scene2v", "2V" ),
        ( "scene3a", "3A" ),
        ( "scene3v", "3V" ),
        ( "scene4a", "4A" ),
        ( "scene4v", "4V" ),
        ( "scene5a", "5A" ),
        ( "scene5v", "5V" ),
        ( "scene6ainside", "6AInside" ),
        ( "scene6aoutside", "6AOutside" ),
        ( "scene6voutside", "6VOutside" ),
        ( "scene7inside", "7Inside" ),
        ( "scene7outside", "7Outside" )
        ]

}

]

# взаимоисключающие теги
define exclusive_exp = [ [ { "naked_yes", "naked_dark_yes", "casual_dress", "casual_blood_dress", "dark_dress" } ], [ { } ] ]

# номер спрайта
define l2di = 0

# не взаимоисключающие теги (ВСЕ! кроме эмоций)
define nonexclusive_exp = [ [ i[0] for i in sum(list(gg_extras[0].values())[1:], [ ]) if i[0] ], [ ] ]

# спрайт тянки
image gg = Live2D("live2d/VampireAvatarV3", default_fade=.5, seamless=True, loop=True, zoom=.3, align=(.5, .5), nonexclusive=nonexclusive_exp[0], attribute_filter=exp_filter)

image ggcow = Live2D("live2d/VampireCowgirl", default_fade=0, seamless=True, loop=True, zoom=1., align=(.5, .5), nonexclusive=nonexclusive_exp[1], attribute_filter=exp_filter)

transform test_align(x=.5, y=1.):
    align (x, y)

transform test_alpha(a=.25):
    alpha a

init -1 python:
    # для копирования текста в буфер
    import pygame.scrap

    # key "ctrl_K_c" action CopyText()
    def copytext(t=None):
        if t is None: 
            t = "show " + gg_test_get_s()

        pygame.scrap.put(pygame.scrap.SCRAP_TEXT, t.encode("utf-8"))
        renpy.notify("The text was copied to the clipboard.")

    CopyText = renpy.curry(copytext)

    # фильтр для взаимоисключения некоторых тегов
    def exp_filter(attrs):
        for a in attrs:
            for e in exclusive_exp:
                if a in e:
                    rv = list(attrs)

                    for remove in e - { a }:
                        while remove in rv: rv.remove(remove)

                    return tuple(rv)

        return attrs

    # получить строку с тегами
    def gg_test_get_s(s=None):
        if s is None: s = l2d_tag[l2di]

        t = " ".join(list(gg_test_s[l2di].values())).strip()

        while "  " in t: t = t.replace("  ", " ")

        if t and s and not s.endswith(" "): s = s + " "

        return s + t

    # показать героиню на слое screens
    def gg_test_show():
        renpy.show(l2d_tag[l2di], what=gg_test_get_s(), layer="screens", at_list=[ test_align ])
        renpy.restart_interaction()

    GGTestShow = renpy.curry(gg_test_show)

    # выбираем теги
    def test_click(key, item):
        store.gg_test_s[l2di][key] = item
        gg_test_show()

    TestClick = renpy.curry(test_click)

    def l2d_prev(plus=-1):
        global l2di

        renpy.hide(l2d_tag[l2di], layer="screens")
        renpy.restart_interaction()
        l2di += plus

        if l2di < 0: l2di = len(l2d_tag) - 1

        if l2di >= len(l2d_tag): l2di = 0

        gg_test_show()

    L2DPrev = renpy.curry(l2d_prev)

    def l2d_next(plus=1):
        l2d_prev(plus)

    L2DNext = renpy.curry(l2d_next)

    def gg_off():
        store.gg_test_s[l2di] = { "emotions": "default" } if l2di == 0 else { "emotions": "" }

    GGOff = renpy.curry(gg_off)

style test_button is textbutton:
    xminimum 480
    padding (20, 5)
    background Transform(gui.idle_color, alpha=.5)
    hover_background gui.accent_color
    selected_background Transform(gui.hover_color, alpha=.5)

style test_button_text is textbutton_text:
    color gui.text_color
    size 24
    xalign .5
    text_align .5
    layout "subtitle"

# запчасти текущей строки для отображения спрайта
default gg_test_s = [ { "emotions": "default" }, { "emotions": "" } ]

# экран для тестирования тегов
screen gg_test_buttons(keys, xa=.0):
    zorder 11

    viewport:
        scrollbars "vertical"
        mousewheel True
        draggable True
        pagekeys True

        xysize (480, config.screen_height - 50)
        align (xa, 1.)

        vbox:
            style_prefix "test"

            for i in list(keys):
                for item, caption in gg_extras[l2di][i]:
                    textbutton caption selected (gg_test_s[l2di].get(i, "") == item) action TestClick(i, item)

                if i != keys[-1]:
                    null height 5

# экран для тестирования спрайта
screen gg_test():
    tag menu

    zorder 10

    timer .2 action GGTestShow(), Trans()

    key "ctrl_K_c" action CopyText()

    style_prefix "test"

    hbox:
        align (.5, .0)
        textbutton "<" action L2DPrev()
        textbutton gg_test_get_s() action CopyText()
        textbutton ">" action L2DNext()

    textbutton "clear" align (.0, .0) xminimum 48 action GGOff(), GGTestShow(), Notify(_("The default state is set."))
    textbutton "x" align (1., .0) xminimum 48 action Function(renpy.hide, l2d_tag[l2di], layer="screens"), Function(renpy.restart_interaction), Return()

    $ kk = list(gg_extras[l2di].keys())

    for keys, xa in [ (kk[:2], .0), (kk[2:], 1.) ]:
        use gg_test_buttons(keys, xa)
