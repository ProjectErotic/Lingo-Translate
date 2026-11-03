image ch_2_paper2:
    contains:
        "ch_2_paper"

    contains:
        Text (_("Спасибо за интерес к нашей игре. В этом письме вы найдете сцену 18+, вырванную из контекста, которая будет вставлена в более подходящий по сюжету момент. Наслаждайтесь."), xysize=(530, 530), color="#000", font="font/pen.ttf")
        pos (525, 190)

image ch_2_paper3:
    contains:
        "ch_2_paper"

    contains:
        Text(_("Спасибо за то, что играли в демо-версию The Blood Diary. Не забудьте добавить игру в вишлист, если хотите продолжить приключение."), xysize=(530, 530), color="#000", font="font/pen.ttf")
        pos (525, 190)

screen letter(t, **kwarg):
    modal True

    if t:
        timer t action End()

    else:
        button style "empty" xfill True yfill True action End()

    for i in kwarg.keys():
        imagebutton idle i at kwarg[i]:
            if i == "photo":
                action End()

transform noat():
    pass

transform uup(y=300, t=1):
    subpixel True
    yoffset y
    ease t yoffset 0

label demo_end:
    $ map_label = ""

    scene kabinet night
    "Вернулась Селена позже, чем я ожидал. Может быть поэтому ощутил ее приближение лишь тогда, когда она уже вошла в поместье."

    scene expression im.Blur(ImageReference("kabinet night"), 5)
    show gg default dark_dress
    with dissolve

    hero "И как это понимать?"
    "Селена сделала невинное лицо."

    show gg default dark_dress talk
    ss "Что именно?"
    hero "Ты ушла, не сказав ни слова. Не думаешь, что стоило меня предупредить?"

    show gg misunderstanding dark_dress talk
    ss "Не припомню, чтобы была здесь узницей. Вполне могу выйти на прогулку без разрешения."
    "Справедливо, но меня такое отношение не устраивает."
    hero "Давай без глупостей. Хочешь ты этого или нет, но у нас теперь общее дело." 

    show gg thoughtfulness dark_dress talk
    ss "Глупости? Мне так и так придётся выходить из этого дома наружу, чтобы это самое \"дело\" выполнить."
    "Эта девчонка..."
    hero "Ты права, но не забывай про мой опыт, которым я могу с тобой поделиться. Тут кругом много опасностей."
    hero "Просто предупреждай меня, когда решишь куда-то выйти, ладно?"

    show gg normal talk dark_dress
    ss "Кроме тебя, ни одной не встретила. Беспокоился обо мне?"
    "Я бросил на нее серьезный взгляд."
    hero "Что, если так?"
    "Девушка нахально улыбнулась."
    hero "Больше так не делай, хорошо? Я должен знать, где ты и что с тобой."

    show gg normal talk dark_dress
    ss "Договорились."
    "Вздохнув, я продолжил."
    hero "Раз ты такая самостоятельная... Для тебя есть особое задание."
    hero "Посмотрим, как ты, дерзкая девчонка, справишься с ним завтра..." 

    scene null with Dissolve (2.0)
    centered "Конец демо-версии."
    centered "В дальнейших событиях вас ожидают первые приключения Селены, а также трудности, радости и... прелести для ее наставника."

    scene expression im.Blur(ImageReference("kabinet night"), 5) with Dissolve (2.0)
    hero "Кажется, у меня на столе какое-то письмо."
    hero "Интересно, что в нем?"

    show screen letter(1, ch_2_letter_closed=uup(1080))
    pause 1

    show screen letter(0, ch_2_letter_bg=noat, ch_2_paper2=uup(200), ch_2_photo=uup(150), ch_2_letter_fg=noat)
    with dissolve
    pause

    menu:
        "- Смотреть 18+ сцену (спойлер) -":
            pass
        "- Не смотреть -":
            jump demo_end_2

    hide screen letter
    scene black 
    with Dissolve (2.0)

    hide screen letter
    show ggcow scene4v
    with Dissolve (2.0)

    $ pause(.5)
    pause

    hide ggcow

    scene black
    with dissolve

    scene kabinet night
    jump demo_end_2

label demo_end_2:
    show screen letter(0, ch_2_letter_bg=noat, ch_2_paper2=noat, ch_2_paper3=uup(200), ch_2_photo=noat, ch_2_letter_fg=noat)
    with dissolve

    $ renpy.pause(5.0, hard=True)

    hide screen letter
    scene black 
    with Dissolve (3.0)
    
    $ renpy.pause(1.0, hard=True)

    call screen follow_story_screen

    return






# follow_screen.rpy

default follow_url = "https://store.steampowered.com/app/4050270/"


screen follow_story_screen():

    modal True
    zorder 100

    # Затемнение всего экрана
    add Solid("#000000aa")

    frame:
        xalign 0.5
        yalign 0.5
        xsize 1080
        ysize 570

        background Frame(Solid("#120d13f2"), 24, 24)
        padding (4, 4)

        # Внешняя тонкая рамка
        frame:
            xfill True
            yfill True
            background Frame(Solid("#3b1b23"), 20, 20)
            padding (2, 2)

            # Основное окно
            frame:
                xfill True
                yfill True
                background Frame(Solid("#171018f5"), 18, 18)
                padding (58, 46)

                vbox:
                    xalign 0.5
                    yalign 0.5
                    spacing 24

                    # Маленький верхний текст
                    text _("История Селены только начинается..."):
                        xalign 0.5
                        textalign 0.5
                        size 30
                        color "#d9a765"
                        outlines [(2, "#000000", 0, 0)]

                    # Декоративная линия
                    add Solid("#7a2c3a88"):
                        xalign 0.5
                        xsize 620
                        ysize 2

                    null height 4

                    # Главный заголовок
                    text _("Хотите следить за её продолжением?"):
                        xalign 0.5
                        textalign 0.5
                        size 46
                        color "#ffffff"
                        outlines [(2, "#000000", 0, 0)]

                    # Описание
                    text _("Добавьте игру в список желаемого и не пропустите новые главы, обновления и продолжение мрачного пути Селены."):
                        xalign 0.5
                        textalign 0.5
                        xmaximum 870
                        size 27
                        color "#d8d0d0"
                        line_spacing 7
                        outlines [(1, "#000000", 0, 0)]

                    null height 16

                    hbox:
                        xalign 0.5
                        spacing 46

                        # Серая, рабочая, но менее привлекательная кнопка
                        button:
                            xsize 255
                            ysize 70

                            background Frame(Solid("#252326dd"), 16, 16)
                            hover_background Frame(Solid("#302d31dd"), 16, 16)
                            activate_sound "audio/click.wav"

                            action [
                                Hide("follow_story_screen"),
                                Return()
                            ]

                            frame:
                                xfill True
                                yfill True
                                background Frame(Solid("#ffffff10"), 14, 14)
                                padding (2, 2)

                                text _("Выход"):
                                    xalign 0.5
                                    yalign 0.5
                                    size 29
                                    color "#8a8588"
                                    hover_color "#aaa3a7"
                                    outlines [(1, "#000000", 0, 0)]

                        # Главная кнопка
                        button:
                            xsize 315
                            ysize 76

                            background Frame(Solid("#a9152f"), 18, 18)
                            hover_background Frame(Solid("#d91f42"), 18, 18)
                            activate_sound "audio/click.wav"

                            action OpenURL(follow_url)

                            frame:
                                xfill True
                                yfill True
                                background Frame(Solid("#ffffff18"), 16, 16)
                                padding (2, 2)

                                text _("Следить"):
                                    xalign 0.5
                                    yalign 0.5
                                    size 32
                                    color "#ffffff"
                                    hover_color "#fff4f4"
                                    outlines [(2, "#000000", 0, 0)]

                    null height 6