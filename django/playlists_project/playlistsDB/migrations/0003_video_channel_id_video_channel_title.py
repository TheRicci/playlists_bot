# Generated by Django 5.0.2 on 2024-03-22 13:08

from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ('playlistsDB', '0002_alter_playlist_video_unique_together_and_more'),
    ]

    operations = [
        migrations.AddField(
            model_name='video',
            name='channel_id',
            field=models.CharField(blank=True, max_length=255, null=True),
        ),
        migrations.AddField(
            model_name='video',
            name='channel_title',
            field=models.CharField(blank=True, max_length=255, null=True),
        ),
    ]